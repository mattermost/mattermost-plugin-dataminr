// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package hashtag

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/biter777/countries"
)

// US state code to full name mapping
var usStates = map[string]string{
	"AL": "Alabama", "AK": "Alaska", "AZ": "Arizona", "AR": "Arkansas",
	"CA": "California", "CO": "Colorado", "CT": "Connecticut", "DE": "Delaware",
	"FL": "Florida", "GA": "Georgia", "HI": "Hawaii", "ID": "Idaho",
	"IL": "Illinois", "IN": "Indiana", "IA": "Iowa", "KS": "Kansas",
	"KY": "Kentucky", "LA": "Louisiana", "ME": "Maine", "MD": "Maryland",
	"MA": "Massachusetts", "MI": "Michigan", "MN": "Minnesota", "MS": "Mississippi",
	"MO": "Missouri", "MT": "Montana", "NE": "Nebraska", "NV": "Nevada",
	"NH": "New Hampshire", "NJ": "New Jersey", "NM": "New Mexico", "NY": "New York",
	"NC": "North Carolina", "ND": "North Dakota", "OH": "Ohio", "OK": "Oklahoma",
	"OR": "Oregon", "PA": "Pennsylvania", "RI": "Rhode Island", "SC": "South Carolina",
	"SD": "South Dakota", "TN": "Tennessee", "TX": "Texas", "UT": "Utah",
	"VT": "Vermont", "VA": "Virginia", "WA": "Washington", "WV": "West Virginia",
	"WI": "Wisconsin", "WY": "Wyoming", "DC": "District of Columbia",
}

// Canadian province/territory code to full name mapping
var canadianProvinces = map[string]string{
	"AB": "Alberta", "BC": "British Columbia", "MB": "Manitoba", "NB": "New Brunswick",
	"NL": "Newfoundland and Labrador", "NS": "Nova Scotia", "NT": "Northwest Territories",
	"NU": "Nunavut", "ON": "Ontario", "PE": "Prince Edward Island", "QC": "Quebec",
	"SK": "Saskatchewan", "YT": "Yukon",
}

// stateZipPattern matches two letters followed by space and a word containing at least one number.
// Examples: "TN 12345", "CA 90210", "NY 10001"
var stateZipPattern = regexp.MustCompile(`(?i)^([A-Za-z]{2})\s+\S*\d\S*$`)

// extractCountryTags extracts location-based hashtags from an address string.
//
// This is the main function that processes location addresses and returns hashtags
// based on the detected location components (city, state/province, country).
//
// Heuristic:
//
// Step 1: Split by commas and clean each part
//   - Drop any part containing numbers (street addresses, zip codes, etc.)
//   - EXCEPTION: "XX 12345" pattern (state code + zip) → keep only "XX"
//     Example: "TN 12345" → "TN", "CA 90210" → "CA"
//
// Step 2: Process based on number of non-empty parts remaining
//
//	Case A - Single part: "San Francisco" or "IL"
//	  - Check if it's a country code/name first
//	  - If country: convert to full country name → "#Israel" (from "IL")
//	  - If not country: use as-is in CamelCase → "#SanFrancisco"
//
//	Case B - Two parts: "San Francisco, California" or "Tel Aviv, IL"
//	  - First part: always create a hashtag in CamelCase → "#SanFrancisco", "#TelAviv"
//	  - Second part (2 chars):
//	    * If it's a US or Canadian state/province code: skip it (ambiguous)
//	      Example: "San Francisco, CA" → "#SanFrancisco" only (CA could be California or Canada)
//	    * If it's NOT a US/Canadian code: try to expand as country
//	      Example: "London, UK" → "#London", "#UnitedKingdom"
//	  - Second part (>2 chars): check if it's a country
//	    If country: use full name → "#TelAviv", "#Israel"
//	    If not country: use as-is → "#SanFrancisco", "#California"
//
//	Case C - Three or more parts: "123 Main St, San Francisco, CA, USA"
//	  - Use ONLY the last 3 parts → "San Francisco, CA, USA"
//	  - Interpret as: City, State/Province, Country
//	  - City: create hashtag in CamelCase → "#SanFrancisco"
//	  - State: expand 2-letter codes for USA/Canada → "#California" (from "CA")
//	  - Country: detect and use full name → "#UnitedStates" (from "USA")
//
// Returns: Array of hashtag strings (e.g., ["#SanFrancisco", "#California", "#UnitedStates"])
//
//	Empty array if all parts are dropped or no valid location found
func extractCountryTags(locationAddress string) []string {
	if locationAddress == "" {
		return nil
	}

	// Step 1: Split and clean parts
	parts := strings.Split(locationAddress, ",")
	var cleanedParts []string

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Check for state + zip code pattern (e.g., "TN 12345")
		if matches := stateZipPattern.FindStringSubmatch(part); matches != nil {
			// Keep only the state code (two letters)
			cleanedParts = append(cleanedParts, matches[1])
			continue
		}

		// Drop parts that contain any numbers
		if containsNumber(part) {
			continue
		}

		cleanedParts = append(cleanedParts, part)
	}

	if len(cleanedParts) == 0 {
		return nil
	}

	// Step 2: Process based on number of parts
	switch len(cleanedParts) {
	case 1:
		return handleSinglePart(cleanedParts[0])
	case 2:
		return handleTwoParts(cleanedParts[0], cleanedParts[1])
	default:
		// Use last 3 parts only
		startIdx := len(cleanedParts) - 3
		return handleThreeParts(cleanedParts[startIdx], cleanedParts[startIdx+1], cleanedParts[startIdx+2])
	}
}

// containsNumber checks if a string contains any digit.
func containsNumber(s string) bool {
	for _, r := range s {
		if unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

// handleSinglePart processes a single location part.
// Checks if it's a country code first, expands to full name if possible.
func handleSinglePart(part string) []string {
	// Always check if it's a country first
	country := detectCountry(part)
	if country != countries.Unknown {
		// Use full country name
		return []string{"#" + camelCase(country.String())}
	}

	// Not a country, just use as-is in CamelCase
	return []string{"#" + camelCase(part)}
}

// handleTwoParts processes two location parts.
// First part always becomes a hashtag.
// Second part: if 2 chars and not a US/Canada state, try to expand as country; if >2 chars, check if country.
func handleTwoParts(first, second string) []string {
	var tags []string

	// First part always becomes a hashtag
	tags = append(tags, "#"+camelCase(first))

	// Second part handling
	if len(second) == 2 {
		// Two-letter code: check if it's a US or Canadian state/province code
		secondUpper := strings.ToUpper(second)
		_, isUSState := usStates[secondUpper]
		_, isCanadianProvince := canadianProvinces[secondUpper]

		if isUSState || isCanadianProvince {
			// Ambiguous - could be a state/province, skip it
			return tags
		}

		// Not a US/Canadian state - check if it's a country code
		country := detectCountry(second)
		if country != countries.Unknown {
			// It's a valid country code, expand to full name
			tags = append(tags, "#"+camelCase(country.String()))
		}
		// If not a country either, skip it
		return tags
	}

	// More than 2 chars: check if it's a country
	country := detectCountry(second)
	if country != countries.Unknown {
		// Use full country name
		tags = append(tags, "#"+camelCase(country.String()))
	} else {
		// Not a country but more than 2 chars, use it as-is
		tags = append(tags, "#"+camelCase(second))
	}

	return tags
}

// handleThreeParts processes three location parts (City, State, Country).
// Expects: City, State/Province, Country
func handleThreeParts(city, state, countryPart string) []string {
	var tags []string

	// Detect the country (last part)
	country := detectCountry(countryPart)

	// City hashtag (third from last)
	tags = append(tags, "#"+camelCase(city))

	// State/Province hashtag (second from last)
	// Try to expand based on detected country
	stateName := expandStateProvince(state, country)
	tags = append(tags, "#"+camelCase(stateName))

	// Country hashtag (last part)
	if country != countries.Unknown {
		tags = append(tags, "#"+camelCase(country.String()))
	} else {
		// Couldn't detect country, use as-is
		tags = append(tags, "#"+camelCase(countryPart))
	}

	return tags
}

// detectCountry tries to identify a country from a string (name or code).
func detectCountry(s string) countries.CountryCode {
	if s == "" {
		return countries.Unknown
	}

	// The ByName method handles both full names and ISO codes (Alpha-2, Alpha-3)
	// It's case-insensitive
	return countries.ByName(s)
}

// expandStateProvince tries to expand a state/province code to full name based on country.
// Only handles USA and Canada.
// Only expands two-letter codes; longer strings are returned as-is.
func expandStateProvince(state string, country countries.CountryCode) string {
	// Only expand if we have a two-letter code
	if len(state) != 2 {
		return state
	}

	stateUpper := strings.ToUpper(state)

	// USA: expand state codes
	if country == countries.US {
		if fullName, exists := usStates[stateUpper]; exists {
			return fullName
		}
	}

	// Canada: expand province/territory codes
	if country == countries.CA {
		if fullName, exists := canadianProvinces[stateUpper]; exists {
			return fullName
		}
	}

	// Not USA/Canada or code not found, return as-is
	return state
}
