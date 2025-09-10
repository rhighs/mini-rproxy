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
	Serial          string `json:"serial"`
	FacilityID      string `json:"facilityId"`
	DeviceType      string `json:"deviceType"`
	ScreenType      string `json:"screenType"`
	OperatingSystem string `json:"operatingSystem"`
	IsKiosk         any    `json:"isKiosk"`
	EquipmentCode   string `json:"equipmentCode"`
	FacilityURL     string `json:"facilityUrl"`
	SWVersion       string `json:"swVersion"`
	Platform        string `json:"platform"`
	MainAppVersion  string `json:"mainAppVersion"`
	DomainID        string `json:"domainId"`
	LOB             string `json:"lob"`
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
	// raw := strings.Join(parts, "|")
	// sig := computeHMACSHA512Hex([]byte(raw+salt), []byte(key))
	// return base64urlEncode([]byte(raw)) + "." + strings.ToUpper(sig)
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

func BuildEquipmentToken(payload *JWTPayload, equipmentContext string) string {
	if payload == nil || payload.EquipmentContext == nil {
		if equipmentContext != "" {
			if b := base64urlDecode(equipmentContext); len(b) > 0 {
				var alt EquipmentContextPayload
				if json.Unmarshal(b, &alt) == nil {
					return buildEquipmentToken(&alt)
				}
			}
		}
	} else {
		if eqToken := buildEquipmentToken(payload.EquipmentContext); eqToken != "" {
			return eqToken
		}
	}
	return ""
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
	return base64urlDecode(parts[1])
}

func base64urlDecode(s string) []byte {
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

func base64urlEncode(b []byte) string {
	if len(b) == 0 {
		// For empty input we can decide on a convention.
		// ASP.NET's UrlTokenEncode returns empty string for empty input.
		// Its decode counterpart returns nil (or empty) accordingly.
		// We'll return just "0" so the decoder can round-trip.
		return "0"
	}

	// Number of padding chars a standard base64 encoding would have added.
	// Formula: (3 - (n % 3)) % 3  -> yields 0,1,2
	pad := (3 - (len(b) % 3)) % 3

	main := base64.RawURLEncoding.EncodeToString(b)
	return main + strconv.Itoa(pad)
}

// RETAINED (for equipment token logic)
func buildEquipmentToken(eq *EquipmentContextPayload) string {
	key := os.Getenv("TGAUTH_LEGACY_PKEY")
	if key == "" {
		return ""
	}

	parts := []string{
		eq.Serial,
		strings.ReplaceAll(eq.FacilityID, "-", ""),
		eq.DeviceType,
		eq.ScreenType,
		eq.OperatingSystem,
		strings.ToLower(asBoolString(eq.IsKiosk)),
		eq.EquipmentCode,
		eq.FacilityURL,
		eq.SWVersion,
		eq.Platform,
		eq.MainAppVersion,
		defaultString(eq.DomainID, "0"),
		defaultString(eq.LOB, "0"),
	}

	raw := strings.Join(parts, "|")
	sig := computeHMACSHA1Hex([]byte(raw), []byte(key))
	return base64urlEncode([]byte(raw)) + "." + strings.ToUpper(sig)
}

func defaultString(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// Generic helpers (some still used by tests or equipment token code)
func asString(v interface{}) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case bool:
		return strconv.FormatBool(t)
	default:
		return ""
	}
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
	switch t := v.(type) {
	case bool:
		return strconv.FormatBool(t)
	case string:
		l := strings.ToLower(t)
		if l == "true" || l == "false" {
			return l
		}
	case float64:
		if t != 0 {
			return "true"
		}
		return "false"
	case int, int8, int16, int32, int64:
		if asInt(t) != 0 {
			return "true"
		}
		return "false"
	}
	return "false"
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
	return trimmed + string('0'+pad)
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
