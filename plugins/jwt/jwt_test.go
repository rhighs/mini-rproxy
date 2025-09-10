package jwt

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func b64Url(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func makeUnsignedJWT(payload any) string {
	h := `{"alg":"none","typ":"JWT"}`
	hEnc := b64Url([]byte(h))
	pb, _ := json.Marshal(payload)
	pEnc := b64Url(pb)
	// empty signature part to satisfy len(parts)==3
	return hEnc + "." + pEnc + "."
}

func withEnv(t *testing.T, k, v string) {
	old, had := os.LookupEnv(k)
	if err := os.Setenv(k, v); err != nil {
		t.Fatalf("set env %s: %v", k, err)
	}
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(k, old)
		} else {
			_ = os.Unsetenv(k)
		}
	})
}

func TestParseJWTPayloadSuccess(t *testing.T) {
	payload := map[string]any{
		"legacycompat": map[string]any{
			"tokenDateTime": "20250905095837",
			"credentialId":  "CRED123",
		},
	}
	token := makeUnsignedJWT(payload)
	got, err := ParseJWTPayload(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.LegacyCompat == nil || got.LegacyCompat.CredentialID != "CRED123" {
		t.Fatalf("unexpected parsed payload: %+v", got)
	}
}

func TestParseJWTPayloadInvalidStructure(t *testing.T) {
	// only two parts -> invalid
	_, err := ParseJWTPayload("abc.def")
	if err == nil {
		t.Fatalf("expected error for invalid jwt structure")
	}
}

func TestBuildLegacyTokenSuccess(t *testing.T) {
	withEnv(t, "TGAUTH_HASH_KEY", "super-secret-key-hex-or-any")
	withEnv(t, "TGAUTH_SIGN_SALT", "pepper-salt")

	lp := &LegacyTokenPayload{
		TokenDateTime:          "2025-09-08T12:34:56Z",
		CredentialID:           "CRED001",
		LanguageID:             "en",
		TimeZoneWindowsID:      "UTC",
		UserCultureName:        "en-US",
		UserID:                 "USR123",
		FacilityURL:            "https://facility.example",
		FacilityID:             "FAC-001",
		StaffID:                "STF-9",
		MeasurementSystem:      1,
		LoginType:              2,
		RequirePasswordOnDev:   nil,
		TokenFor:               0, // => Professional
		IntegrationAPIKey:      "INTKEY",
		IsChainPosition:        nil,
		ClientApplicationTypes: nil,
		IterationNumber:        5,
		SlidingValue:           999,
		Domain:                 "example.com",
	}

	jp := &JWTPayload{LegacyCompat: lp}
	got := BuildLegacyToken(jp)
	if got == "" {
		t.Fatalf("expected non-empty legacy token")
	}

	// Independently reconstruct expected token
	applicationType := "Professional"
	if asInt(lp.TokenFor) == 1 {
		applicationType = "EndUser"
	} else if asInt(lp.TokenFor) != 0 {
		applicationType = "MobileUserWebApp"
	}
	yodaAppID := map[string]string{
		"EndUser":        "EC1D38D7-D359-48D0-A60C-D8C0B8FB9DF9",
		"Professional":   "69295ED5-A53C-434B-8518-F2E0B5F05B28",
		"Equipment":      "9143E6D6-F36A-44E8-AE8C-4698EA897557",
		"Integration":    "F41CEC0F-6B5B-4AFF-89B3-B22CD144DD7E",
		"TechnogymAdmin": "AC429D52-7860-4149-AF02-495A05306EA6",
		"MyWellnessLink": "9FACBB1F-7B37-431f-947D-555797110319",
		"OAuth":          "58FB87D2-B9C1-45D1-83CE-F92C64E787AF",
	}
	appID := yodaAppID[applicationType]

	parts := []string{
		lp.TokenDateTime,
		lp.CredentialID,
		appID,
		asString(lp.LanguageID),
		lp.TimeZoneWindowsID,
		lp.UserCultureName,
		lp.UserID,
		lp.FacilityURL,
		lp.FacilityID,
		lp.StaffID,
		asString(lp.MeasurementSystem),
		asString(lp.LoginType),
		asString(lp.RequirePasswordOnDev),
		asString(lp.TokenFor),
		lp.IntegrationAPIKey,
		asString(lp.IsChainPosition),
		asString(lp.ClientApplicationTypes),
		asString(lp.IterationNumber),
		asString(lp.SlidingValue),
		lp.Domain,
	}
	raw := strings.Join(parts, "|")
	key := os.Getenv("TGAUTH_HASH_KEY")
	salt := os.Getenv("TGAUTH_SIGN_SALT")
	expected := base64urlEncode([]byte(raw)) + "." + strings.ToUpper(computeHMACSHA512Hex([]byte(raw+salt), []byte(key)))

	if got != expected {
		t.Fatalf("legacy token mismatch\n got: %s\nexp: %s", got, expected)
	}
}

func TestBuildLegacyTokenMissingEnv(t *testing.T) {
	_ = os.Unsetenv("TGAUTH_HASH_KEY")
	_ = os.Unsetenv("TGAUTH_SIGN_SALT")
	lp := &LegacyTokenPayload{TokenDateTime: "x"}
	jp := &JWTPayload{LegacyCompat: lp}
	if tok := BuildLegacyToken(jp); tok != "" {
		t.Fatalf("expected empty token when env vars missing, got %s", tok)
	}
}

func TestBuildEquipmentTokenFromPayload(t *testing.T) {
	withEnv(t, "TGAUTH_LEGACY_PKEY", "equip-key-123")

	eq := &EquipmentContextPayload{
		Serial:          "SER123",
		FacilityID:      "FAC-XYZ",
		DeviceType:      "treadmill",
		ScreenType:      "lcd",
		OperatingSystem: "linux",
		IsKiosk:         true,
		EquipmentCode:   "EQCODE",
		FacilityURL:     "https://facility.example",
		SWVersion:       "1.0.0",
		Platform:        "platformX",
		MainAppVersion:  "2.3.4",
		DomainID:        "42",
		LOB:             "7",
	}
	jp := &JWTPayload{EquipmentContext: eq}
	got := BuildEquipmentToken(jp, "")
	if got == "" {
		t.Fatalf("expected non-empty equipment token")
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
	key := os.Getenv("TGAUTH_LEGACY_PKEY")
	expected := base64urlEncode([]byte(raw)) + "." + strings.ToUpper(computeHMACSHA1Hex([]byte(raw), []byte(key)))

	if got != expected {
		t.Fatalf("equipment token mismatch\n got: %s\nexp: %s", got, expected)
	}
}

func TestBuildEquipmentTokenFromContextParam(t *testing.T) {
	withEnv(t, "TGAUTH_LEGACY_PKEY", "equip-key-abc")

	eq := EquipmentContextPayload{
		Serial:          "SER777",
		FacilityID:      "FAC-777",
		DeviceType:      "bike",
		ScreenType:      "oled",
		OperatingSystem: "android",
		IsKiosk:         false,
		EquipmentCode:   "EQ777",
		FacilityURL:     "https://f.example",
		SWVersion:       "9.9.9",
		Platform:        "android",
		MainAppVersion:  "5.6.7",
		DomainID:        "",
		LOB:             "",
	}
	// JWTPayload has nil EquipmentContext, so it must fall back to provided context
	jp := &JWTPayload{}

	js, _ := json.Marshal(eq)
	ctxParam := base64.RawURLEncoding.EncodeToString(js)
	got := BuildEquipmentToken(jp, ctxParam)
	if got == "" {
		t.Fatalf("expected non-empty equipment token from context param")
	}
}

func TestCLIIntegrationLegacyFromJWT(t *testing.T) {
	withEnv(t, "TGAUTH_HASH_KEY", "cli-secret-key")
	withEnv(t, "TGAUTH_SIGN_SALT", "cli-salt")

	// Build a JWT with legacycompat
	lp := map[string]any{
		"tokenDateTime":     "20250905095837",
		"credentialId":      "CLI-CRED",
		"languageId":        "en",
		"timeZoneWindowsId": "UTC",
		"userCultureName":   "en-US",
		"userId":            "USERCLI",
		"facilityUrl":       "https://cli.fac",
		"facilityId":        "FACCLI",
		"staffId":           "STFCLI",
		"measurementSystem": 1,
		"loginType":         2,
		"tokenFor":          0,
		"integrationApiKey": "X",
		"iterationNumber":   1,
		"slidingValue":      2,
		"domain":            "domain.cli",
	}
	payload := map[string]any{"legacycompat": lp}
	jwtStr := makeUnsignedJWT(payload)

	cmd := exec.Command("go", "run", "./cmd", "legacy-from-jwt", "--jwt", jwtStr)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cli error: %v output=%s", err, string(out))
	}
	token := strings.TrimSpace(string(out))
	if !strings.Contains(token, ".") {
		t.Fatalf("expected legacy token form, got: %s", token)
	}
}

func TestCLIIntegrationLegacyBuild(t *testing.T) {
	withEnv(t, "TGAUTH_HASH_KEY", "cli-secret-key2")
	withEnv(t, "TGAUTH_SIGN_SALT", "cli-salt2")

	args := []string{
		"legacy-build",
		"tokenDateTime=20250905095837",
		"credentialId=CRED2",
		"languageId=en",
		"timeZoneWindowsId=UTC",
		"userCultureName=en-US",
		"userId=USR2",
		"facilityUrl=https://fac2",
		"facilityId=FAC2",
		"staffId=STF2",
		"measurementSystem=1",
		"loginType=2",
		"tokenFor=1",
		"integrationApiKey=IK2",
		"iterationNumber=1",
		"slidingValue=2",
		"domain=dom2",
	}
	cmd := exec.Command("go", append([]string{"run", "./cmd"}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cli error: %v output=%s", err, string(out))
	}
	token := strings.TrimSpace(string(out))
	if !strings.Contains(token, ".") {
		t.Fatalf("expected legacy token format from build, got %s", token)
	}
}

func TestCLIParseJWT(t *testing.T) {
	payload := map[string]any{
		"legacycompat": map[string]any{
			"credentialId": "PARSEME",
		},
	}
	jwtStr := makeUnsignedJWT(payload)
	cmd := exec.Command("go", "run", "./cmd", "parse", "--jwt", jwtStr)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cli error: %v output=%s", err, string(out))
	}
	if !strings.Contains(string(out), "PARSEME") {
		t.Fatalf("expected parsed JSON to contain credentialId PARSEME, got %s", out)
	}
}
