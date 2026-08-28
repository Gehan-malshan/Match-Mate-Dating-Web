package domain

import (
	"net/mail"
	"regexp"
	"sort"
	"strings"
	"time"
)

var contactPattern = regexp.MustCompile(`(?i)(https?://|www\.|[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}|(?:\+?\d[\d\s().-]{7,}\d)|(?:instagram|facebook|whatsapp|telegram|snapchat|tiktok|linkedin)\s*[:@])`)

func NormalizeEmail(v string) (string, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	a, err := mail.ParseAddress(v)
	if err != nil || a.Address != v || len(v) > 254 {
		return "", &ProblemError{400, "ACCOUNT_EMAIL_INVALID", "Invalid email", "Enter a valid email address.", map[string]string{"email": "invalid"}}
	}
	return v, nil
}
func ValidateRegistration(v RegisterInput, minimum int, now time.Time) map[string]string {
	fields := map[string]string{}
	if _, err := NormalizeEmail(v.Email); err != nil {
		fields["email"] = "invalid"
	}
	if ValidatePassword(v.Password) != nil {
		fields["password"] = "must contain 12 to 128 characters"
	}
	if ValidateProfileText(v.Nickname, "", "") != nil {
		fields["nickname"] = "must contain 2 to 40 safe characters"
	}
	if ValidateAdult(v.DateOfBirth, minimum, now) != nil {
		fields["dateOfBirth"] = "must be a valid adult date"
	}
	if strings.TrimSpace(v.ConsentVersion) == "" {
		fields["consentVersion"] = "required"
	}
	return fields
}
func ValidateProfilePatch(v ProfilePatch, minimum int, now time.Time) map[string]string {
	fields := map[string]string{}
	nickname, location, bio := "member", "", ""
	if v.Nickname != nil {
		nickname = *v.Nickname
	}
	if v.BroadLocation != nil {
		location = *v.BroadLocation
	}
	if v.Bio != nil {
		bio = *v.Bio
	}
	if ValidateProfileText(nickname, location, bio) != nil {
		fields["profile"] = "contains invalid text or contact information"
	}
	if v.DateOfBirth != nil && ValidateAdult(*v.DateOfBirth, minimum, now) != nil {
		fields["dateOfBirth"] = "must be a valid adult date"
	}
	if v.Visibility != nil && *v.Visibility != VisibilityPrivate && *v.Visibility != VisibilityCommunity {
		fields["visibility"] = "must be PRIVATE or COMMUNITY"
	}
	if v.Interests != nil {
		if _, err := NormalizeList(*v.Interests, 20, 50); err != nil {
			fields["interests"] = "contains invalid values"
		}
	}
	return fields
}
func ValidatePassword(v string) error {
	if len(v) < 12 || len(v) > 128 {
		return &ProblemError{400, "ACCOUNT_PASSWORD_INVALID", "Invalid password", "Password must contain 12 to 128 characters.", map[string]string{"password": "length"}}
	}
	return nil
}
func ValidateAdult(dob string, minimum int, now time.Time) error {
	d, err := time.Parse("2006-01-02", dob)
	if err != nil {
		return &ProblemError{400, "PROFILE_DOB_INVALID", "Invalid date of birth", "Use YYYY-MM-DD.", map[string]string{"dateOfBirth": "invalid"}}
	}
	cutoff := now.UTC().AddDate(-minimum, 0, 0)
	if d.After(cutoff) {
		return &ProblemError{400, "ACCOUNT_MINIMUM_AGE", "Minimum age not met", "You must meet the minimum age requirement.", map[string]string{"dateOfBirth": "minimum_age"}}
	}
	if d.Before(now.UTC().AddDate(-120, 0, 0)) {
		return &ProblemError{400, "PROFILE_DOB_INVALID", "Invalid date of birth", "Enter a valid date of birth.", map[string]string{"dateOfBirth": "range"}}
	}
	return nil
}
func ValidateProfileText(nickname, location, bio string) error {
	nickname = strings.TrimSpace(nickname)
	location = strings.TrimSpace(location)
	bio = strings.TrimSpace(bio)
	if len(nickname) < 2 || len(nickname) > 40 {
		return &ProblemError{400, "PROFILE_NICKNAME_INVALID", "Invalid nickname", "Nickname must contain 2 to 40 characters.", map[string]string{"nickname": "length"}}
	}
	if len(location) > 80 || len(bio) > 500 {
		return &ProblemError{400, "PROFILE_TEXT_INVALID", "Profile text is too long", "Review the profile field limits.", nil}
	}
	if contactPattern.MatchString(nickname + " " + location + " " + bio) {
		return &ProblemError{400, "PROFILE_CONTACT_INFO_NOT_ALLOWED", "Contact information is not allowed", "Remove contact details, links, or social handles from community profile fields.", nil}
	}
	return nil
}
func NormalizeList(values []string, maxItems, maxLength int) ([]string, error) {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if len(v) > maxLength || contactPattern.MatchString(v) {
			return nil, &ProblemError{400, "PROFILE_VALUE_INVALID", "Invalid profile value", "A profile value is too long or contains contact information.", nil}
		}
		k := strings.ToLower(v)
		if !seen[k] {
			seen[k] = true
			out = append(out, v)
		}
	}
	if len(out) > maxItems {
		return nil, &ProblemError{400, "PROFILE_VALUE_INVALID", "Too many values", "Reduce the number of selected values.", nil}
	}
	sort.Strings(out)
	return out, nil
}
func AgeBand(dob string, now time.Time) string {
	d, err := time.Parse("2006-01-02", dob)
	if err != nil {
		return ""
	}
	age := now.Year() - d.Year()
	if now.YearDay() < d.YearDay() {
		age--
	}
	low := (age / 5) * 5
	if low < 18 {
		low = 18
	}
	return strings.TrimSpace(strings.Join([]string{itoa(low), itoa(low + 4)}, "-"))
}
func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	b := make([]byte, 0, 4)
	for v > 0 {
		b = append([]byte{byte('0' + v%10)}, b...)
		v /= 10
	}
	return string(b)
}
