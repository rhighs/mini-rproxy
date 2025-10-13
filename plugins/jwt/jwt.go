package jwt

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type LegacyTokenPayload struct {
	TokenDateTime          string      `json:"tokenDateTime"`
	ApplicationID          string      `json:"applicationId"`
	CredentialID           string      `json:"credentialId"`
	LanguageID             interface{} `json:"languageId"`
	TimeZoneWindowsID      string      `json:"timeZoneWindowsId"`
	UserCultureName        string      `json:"userCultureName"`
	UserID                 string      `json:"userId"`
	FacilityURL            string      `json:"facilityUrl"`
	FacilityID             string      `json:"facilityId"`
	StaffID                string      `json:"staffId"`
	MeasurementSystem      interface{} `json:"measurementSystem"`
	LoginType              interface{} `json:"loginType"`
	RequirePasswordOnDev   interface{} `json:"requirePasswordOnDevice"`
	TokenFor               interface{} `json:"tokenFor"`
	IntegrationAPIKey      string      `json:"integrationApiKey"`
	IsChainPosition        interface{} `json:"isChainPosition"`
	ClientApplicationTypes interface{} `json:"clientApplicationTypes"`
	IterationNumber        interface{} `json:"iterationNumber"`
	SlidingValue           interface{} `json:"slidingValue"`
	Domain                 string      `json:"domain"`
}

type EquipmentContextPayload struct {
	Serial          string `json:"s"`
	FacilityID      string `json:"fid"`
	DeviceType      string `json:"dt"`
	ScreenType      string `json:"st"`
	OperatingSystem string `json:"os"`
	IsKiosk         any    `json:"k"`
	EquipmentCode   string `json:"ec"`
	FacilityURL     string `json:"furl"`
	SWVersion       string `json:"v"`
	Platform        string `json:"p"`
}

type JWTPayload struct {
	LegacyCompat     *LegacyTokenPayload      `json:"legacycompat"`
	EquipmentContext *EquipmentContextPayload `json:"equipmentContext"`
}

// BuildLegacyToken replicates the legacy (C#) token construction semantics:
// - GUIDs formatted with "N" (32 lowercase hex, no hyphens)
// - Booleans mapped to "1"/"0" (RequirePasswordOnDevice) or "1"/"" (IsChain)
// - Enum / numeric fields rendered using invariant culture (decimal digits)
// - Domain forced to lowercase
// - Order of 20 fields strictly preserved
func BuildLegacyToken(payload *JWTPayload) string {
	if payload == nil || payload.LegacyCompat == nil {
		return ""
	}
	l := payload.LegacyCompat

	key := os.Getenv("TGAUTH_HASH_KEY")
	salt := os.Getenv("TGAUTH_SIGN_SALT")
	if key == "" || salt == "" {
		return ""
	}

	parts := []string{
		strings.TrimSpace(l.TokenDateTime),
		guidN(l.CredentialID),
		strings.TrimSpace(l.ApplicationID),
		intLikeString(l.LanguageID),
		l.TimeZoneWindowsID,
		l.UserCultureName,
		guidN(l.UserID),
		l.FacilityURL,
		guidN(l.FacilityID),
		guidN(l.StaffID),
		intLikeString(l.MeasurementSystem),
		intLikeString(l.LoginType),
		bool10String(l.RequirePasswordOnDev), // "1" or "0"
		intLikeString(l.TokenFor),            // numeric short
		l.IntegrationAPIKey,
		bool1EmptyString(l.IsChainPosition),     // "1" or ""
		intLikeString(l.ClientApplicationTypes), // numeric short
		intLikeString(l.IterationNumber),
		intLikeString(l.SlidingValue),
		strings.ToLower(l.Domain),
	}

	return EncodeUserToken(parts, []byte(key), salt)
}

func guidN(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	return strings.ReplaceAll(s, "-", "")
}

func intLikeString(v interface{}) string {
	switch t := v.(type) {
	case nil:
		return ""
	case int:
		return strconv.Itoa(t)
	case int8, int16, int32, int64:
		return fmt.Sprintf("%d", t)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", t)
	case float32, float64:
		// Legacy code used short/int; treat floats with whole values as int
		f := asInt(v)
		return strconv.FormatInt(f, 10)
	case string:
		// Accept already numeric string
		if t == "" {
			return ""
		}
		if _, err := strconv.ParseInt(t, 10, 64); err == nil {
			return t
		}
		return t // fallback (languageId might have been symbolic historically)
	case bool:
		if t {
			return "1"
		}
		return "0"
	default:
		return ""
	}
}

func bool10String(v interface{}) string {
	switch t := v.(type) {
	case bool:
		if t {
			return "1"
		}
		return "0"
	case string:
		lt := strings.ToLower(strings.TrimSpace(t))
		if lt == "1" || lt == "true" {
			return "1"
		}
		return "0"
	case int, int64, int32, int16, int8:
		if asInt(t) != 0 {
			return "1"
		}
		return "0"
	case float32, float64:
		if asInt(t) != 0 {
			return "1"
		}
		return "0"
	}
	return "0"
}

func bool1EmptyString(v interface{}) string {
	switch t := v.(type) {
	case bool:
		if t {
			return "1"
		}
	case string:
		lt := strings.ToLower(strings.TrimSpace(t))
		if lt == "1" || lt == "true" {
			return "1"
		}
	case int, int64, int32, int16, int8:
		if asInt(t) != 0 {
			return "1"
		}
	case float32, float64:
		if asInt(t) != 0 {
			return "1"
		}
	}
	return ""
}

func EquipmentTokenFromPayload(payload *JWTPayload) (string, error) {
	eqToken, err := buildEquipmentToken(payload.EquipmentContext)
	if err != nil {
		return "", err
	}

	return eqToken, nil
}

func EquipmentTokenFromContext(equipmentContext string) (string, error) {
	b := infernoDecode(equipmentContext)
	var alt EquipmentContextPayload
	if err := json.Unmarshal(b, &alt); err != nil {
		return "", errors.New("malformed json payload, got error: " + err.Error())
	}

	token, err := buildEquipmentToken(&alt)
	if err != nil {
		return "", err
	}

	return token, nil
}

func ParseJWTPayload(token string) (*JWTPayload, error) {
	payloadJSON := decodeJWTPayload(token)
	if payloadJSON == nil {
		return nil, errors.New("invalid jwt structure or payload decode failed")
	}

	var payload JWTPayload
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal jwt payload: %w", err)
	}

	return &payload, nil
}

func decodeJWTPayload(jwt string) []byte {
	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		return nil
	}
	return infernoDecode(parts[1])
}

func infernoDecode(s string) []byte {
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	s = strings.ReplaceAll(s, "-", "+")
	s = strings.ReplaceAll(s, "_", "/")
	b, _ := base64.StdEncoding.DecodeString(s)
	return b
}

func infernoEncode(b []byte) string {
	s := base64.StdEncoding.EncodeToString(b)
	pad := 0
	for len(s) > 0 && s[len(s)-1] == '=' {
		pad++
		s = s[:len(s)-1]
	}
	s = strings.NewReplacer("+", "-", "/", "_").Replace(s)
	return s + strconv.Itoa(pad)
}

func buildEquipmentToken(eq *EquipmentContextPayload) (string, error) {
	key := os.Getenv("TGAUTH_LEGACY_PKEY")
	if key == "" {
		return "", errors.New("TGAUTH_LEGACY_PKEY env must be set")
	}

	parts := []string{
		eq.Serial,
		strings.ReplaceAll(eq.FacilityID, "-", ""),
		eq.DeviceType,
		eq.ScreenType,
		eq.OperatingSystem,
		asBoolString(eq.IsKiosk),
		eq.EquipmentCode,
		eq.FacilityURL,
		eq.SWVersion,
		eq.Platform,

		// NOTE(rob): ||0|1 are for defaults for:
		//
		//  MainAppVersion = tokenElements.Length >= 11
		//                    ? tokenElements[10]
		//                    : null,
		//  DomainId = tokenElements.Length >= 12
		//                    ? Convert.ToByte(tokenElements[11])
		//                    : (byte)0,
		//  Lob = tokenElements.Length >= 13
		//                    ? tokenElements[12].ToEnum<LobTypes>()
		//                    : LobTypes.Home
		//
		// this stuff can be found here:
		// https://github.com/tgym-digital/technogym.mwcloud.packages/blob/afe1a230584db286e761c7cad952572b0688ff51/src/Security/Technogym.MwCloud.Security.Token/Implementations/EquipmentTokenService.cs#L29
		//
		"",
		"0",
		"1",
	}

	raw := strings.Join(parts, "|")
	bytes, err := hex.DecodeString(key)
	if err != nil {
		return "", err
	}

	sig := computeHMACSHA1Hex([]byte(raw), bytes)
	token := infernoEncode([]byte(raw)) + "." + strings.ToUpper(sig)

	return token, nil
}

func asInt(v interface{}) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case int:
		return int64(t)
	case int64:
		return t
	case int32:
		return int64(t)
	case int16:
		return int64(t)
	case int8:
		return int64(t)
	case string:
		if i, err := strconv.ParseInt(t, 10, 64); err == nil {
			return i
		}
	case bool:
		if t {
			return 1
		}
		return 0
	}
	return 0
}

func asBoolString(v interface{}) string {
	if asInt(v) == 1 {
		return "True"
	} else {
		return "False"
	}
}

func computeHMACSHA512Hex(message, secret []byte) string {
	mac := hmac.New(sha512.New, secret)
	_, _ = mac.Write(message)
	sum := mac.Sum(nil)
	return hex.EncodeToString(sum)
}

func urlTokenEncode(input []byte) string {
	if len(input) == 0 {
		return ""
	}
	s := base64.StdEncoding.EncodeToString(input)
	// remove padding and record padding length in last char
	pad := 0
	for i := len(s) - 1; i >= 0 && s[i] == '='; i-- {
		pad++
	}
	trimmed := s[:len(s)-pad]
	trimmed = strings.ReplaceAll(trimmed, "+", "-")
	trimmed = strings.ReplaceAll(trimmed, "/", "_")
	// append a single digit char equal to original padding count
	return trimmed + strconv.Itoa(pad)
}

func hashString512(key []byte, s string) string {
	mac := hmac.New(sha512.New, key)
	mac.Write([]byte(s))
	sum := mac.Sum(nil)
	return strings.ToUpper(hex.EncodeToString(sum))
}

func EncodeUserToken(tokenElements []string, Hashkey512 []byte, Salt string) string {
	text := strings.Join(tokenElements, "|")
	tokenPart := urlTokenEncode([]byte(text))
	hashPart := hashString512(Hashkey512, text+Salt)
	return tokenPart + "." + hashPart
}

func computeHMACSHA1Hex(message, secret []byte) string {
	mac := hmac.New(sha1.New, secret)
	_, _ = mac.Write(message)
	sum := mac.Sum(nil)
	return hex.EncodeToString(sum)
}
