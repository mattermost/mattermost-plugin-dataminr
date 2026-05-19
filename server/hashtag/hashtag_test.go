// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package hashtag

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mattermost/mattermost-plugin-dataminr/server/backend"
)

func TestGenerate_ComplexMultiCategoryTopics(t *testing.T) {
	// Test with complex multi-category topics matching Python test
	alert := backend.Alert{
		AlertType: "Flash",
		Location: &backend.Location{
			Address: "Kyiv, Ukraine",
		},
		Topics: []string{
			"Conflicts - Air",
			"Conflicts - Land",
			"Transportation - Aviation",
			"Disasters and Weather - Search and Rescue",
			"Disasters and Weather - Structure Fires and Collapses",
		},
	}

	result := Generate(alert)
	// With new logic: All segments treated equally
	// "Conflicts - Air" → #Conflicts, #Air
	// "Disasters and Weather - Search and Rescue" → #Disasters, #Weather, #DisastersAndWeather, #Search, #Rescue, #SearchAndRescue
	// "Disasters and Weather - Structure Fires and Collapses" → #Disasters, #Weather, #DisastersAndWeather (all deduped), #Structure, #Fires, #Collapses
	expected := "🏷️ #Flash, #Kyiv, #Ukraine, #Conflicts, #Air, #Land, #Transportation, #Aviation, #Disasters, #Weather, #DisastersAndWeather, #Search, #Rescue, #SearchAndRescue, #Structure, #Fires, #Collapses"

	assert.Equal(t, expected, result, "Complex topics should be parsed correctly")
}

func TestGenerate_USALocation(t *testing.T) {
	// Test with USA location - CA should be expanded to California
	alert := backend.Alert{
		AlertType: "Urgent",
		Location: &backend.Location{
			Address: "San Francisco, CA, USA",
		},
		Topics: []string{
			"Fire",
			"Emergency Response",
		},
	}

	result := Generate(alert)

	// Should have #Urgent, #SanFrancisco, #California, #UnitedStates, #Fire, #Emergency, #Response
	assert.Contains(t, result, "#Urgent")
	assert.Contains(t, result, "#SanFrancisco")
	assert.Contains(t, result, "#California")
	assert.Contains(t, result, "#UnitedStates")
	assert.Contains(t, result, "#Fire")
	assert.Contains(t, result, "#Emergency")
	assert.Contains(t, result, "#Response")
}

func TestGenerate_NoLocation(t *testing.T) {
	// Test without location data
	alert := backend.Alert{
		AlertType: "Alert",
		Topics: []string{
			"Infrastructure",
			"Power Outage",
		},
	}

	result := Generate(alert)
	// "Power Outage" is 2 words (no "and"), so creates #Power, #Outage
	expected := "🏷️ #Alert, #Infrastructure, #Power, #Outage"

	assert.Equal(t, expected, result, "Should work without location")
}

func TestGenerate_Deduplication(t *testing.T) {
	// Test deduplication of repeated tags
	alert := backend.Alert{
		AlertType: "Flash",
		Location: &backend.Location{
			Address: "London, United Kingdom",
		},
		Topics: []string{
			"Fire - Structure Fire",
			"Fire - Wildfire",
			"Emergency Response",
		},
	}

	result := Generate(alert)

	// With new logic:
	// "Fire - Structure Fire" → #Fire, #Structure, #Fire (deduped to one #Fire)
	// "Fire - Wildfire" → #Fire (deduped), #Wildfire
	// "Emergency Response" → #Emergency, #Response
	assert.Contains(t, result, "#Flash")
	assert.Contains(t, result, "#London")
	assert.Contains(t, result, "#UnitedKingdom")
	assert.Contains(t, result, "#Fire")
	assert.Contains(t, result, "#Structure")
	assert.Contains(t, result, "#Wildfire")
	assert.Contains(t, result, "#Emergency")
	assert.Contains(t, result, "#Response")

	// Count occurrences of #Fire - should only appear once due to deduplication
	fireCount := 0
	for i := 0; i < len(result)-4; i++ {
		if result[i:i+5] == "#Fire" {
			// Make sure it's not part of another word like #Wildfire
			if i+5 >= len(result) || (result[i+5] == ',' || result[i+5] == ' ') {
				fireCount++
			}
		}
	}
	assert.Equal(t, 1, fireCount, "Fire should only appear once despite being in multiple topics")
}

func TestGenerate_ThreeWordAndPhrases(t *testing.T) {
	// Test that 3-word phrases with "and" in middle create both individual and combined tags
	alert := backend.Alert{
		AlertType: "Urgent",
		Topics: []string{
			"Command and Control",         // 3 words - creates all variants
			"Search and Rescue",           // 3 words - creates all variants
			"Fire and Emergency Services", // 4 words - only individual words
		},
	}

	result := Generate(alert)

	// 3-word phrases should have individual words AND combined version
	assert.Contains(t, result, "#Command")
	assert.Contains(t, result, "#Control")
	assert.Contains(t, result, "#CommandAndControl", "Command and Control should create combined tag")

	assert.Contains(t, result, "#Search")
	assert.Contains(t, result, "#Rescue")
	assert.Contains(t, result, "#SearchAndRescue", "Search and Rescue should create combined tag")

	// 4-word phrase should only have individual words
	assert.Contains(t, result, "#Fire")
	assert.Contains(t, result, "#Emergency")
	assert.Contains(t, result, "#Services")
	// Should NOT have combined version (more than 3 words)
	assert.NotContains(t, result, "#FireAndEmergencyServices")
}

func TestGenerate_EmptyAlert(t *testing.T) {
	// Test with minimal alert data
	alert := backend.Alert{}

	result := Generate(alert)
	expected := "🏷️ #Alert" // Default alert type

	assert.Equal(t, expected, result, "Should handle empty alert with default type")
}

func TestGenerate_OnlyAlertType(t *testing.T) {
	// Test with only alert type
	alert := backend.Alert{
		AlertType: "Flash",
	}

	result := Generate(alert)
	expected := "🏷️ #Flash"

	assert.Equal(t, expected, result, "Should work with only alert type")
}

func TestGenerate_MultipleCountries(t *testing.T) {
	// Test that with 4+ parts, we use last 3 (City, State, Country)
	alert := backend.Alert{
		AlertType: "Alert",
		Location: &backend.Location{
			Address: "Border region, France, Germany, Switzerland",
		},
	}

	result := Generate(alert)

	// With the new logic, we take last 3 parts: France, Germany, Switzerland
	// Interpreted as: City=France, State=Germany, Country=Switzerland
	assert.Contains(t, result, "#France")
	assert.Contains(t, result, "#Germany")
	assert.Contains(t, result, "#Switzerland")
}

func TestExtractAlertLevelTag(t *testing.T) {
	tests := []struct {
		name      string
		alertType string
		expected  string
	}{
		{"Flash alert", "Flash", "#Flash"},
		{"Urgent alert", "Urgent", "#Urgent"},
		{"Alert alert", "Alert", "#Alert"},
		{"Lowercase flash", "flash", "#Flash"},
		{"Uppercase URGENT", "URGENT", "#Urgent"},
		{"Empty alert type", "", "#Alert"},
		{"Whitespace", "  Flash  ", "#Flash"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractAlertLevelTag(tt.alertType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractCountryTags(t *testing.T) {
	tests := []struct {
		name     string
		address  string
		expected []string
	}{
		// Single part scenarios
		{
			name:     "Single part - country code",
			address:  "IL",
			expected: []string{"#Israel"},
		},
		{
			name:     "Single part - city name",
			address:  "San Francisco",
			expected: []string{"#SanFrancisco"},
		},
		{
			name:     "Single part - country name",
			address:  "Ukraine",
			expected: []string{"#Ukraine"},
		},

		// Two part scenarios
		{
			name:     "Two parts - city and 2-char code (skip second)",
			address:  "San Francisco, CA",
			expected: []string{"#SanFrancisco"},
		},
		{
			name:     "Two parts - city and country name",
			address:  "Tel Aviv, Israel",
			expected: []string{"#TelAviv", "#Israel"},
		},
		{
			name:     "Two parts - city and 2-char country code (not US/CA state)",
			address:  "London, UK",
			expected: []string{"#London", "#UnitedKingdom"},
		},
		{
			name:     "Two parts - city and region (not country)",
			address:  "San Francisco, California",
			expected: []string{"#SanFrancisco", "#California"},
		},

		// Three+ part scenarios
		{
			name:     "Three parts - USA with state expansion",
			address:  "San Francisco, CA, USA",
			expected: []string{"#SanFrancisco", "#California", "#UnitedStates"},
		},
		{
			name:     "Three parts - Canada with province expansion",
			address:  "Toronto, ON, Canada",
			expected: []string{"#Toronto", "#Ontario", "#Canada"},
		},
		{
			name:     "Three parts - country code",
			address:  "Toronto, ON, CA",
			expected: []string{"#Toronto", "#Ontario", "#Canada"},
		},
		{
			name:     "Three parts - with street address (dropped)",
			address:  "123 Main St, Nashville, TN, USA",
			expected: []string{"#Nashville", "#Tennessee", "#UnitedStates"},
		},
		{
			name:     "Three parts - with zip code pattern",
			address:  "Nashville, TN 37203, USA",
			expected: []string{"#Nashville", "#Tennessee", "#UnitedStates"},
		},
		{
			name:     "Four parts - use last 3",
			address:  "Downtown, San Francisco, CA, USA",
			expected: []string{"#SanFrancisco", "#California", "#UnitedStates"},
		},
		{
			name:     "Five parts with numbers - use last 3 non-numeric",
			address:  "123 Main St, Suite 456, Nashville, TN, USA",
			expected: []string{"#Nashville", "#Tennessee", "#UnitedStates"},
		},

		// Edge cases
		{
			name:     "Empty address",
			address:  "",
			expected: nil,
		},
		{
			name:     "Only numbers (all dropped)",
			address:  "12345, 67890",
			expected: nil,
		},
		{
			name:     "Unknown country code",
			address:  "City, XX, YY",
			expected: []string{"#City", "#XX", "#YY"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractCountryTags(tt.address)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractTopicTags(t *testing.T) {
	tests := []struct {
		name     string
		topics   []string
		expected []string
	}{
		{
			name:     "Simple topic",
			topics:   []string{"Fire"},
			expected: []string{"#Fire"},
		},
		{
			name:     "Category with subcategory",
			topics:   []string{"Conflicts - Air"},
			expected: []string{"#Conflicts", "#Air"},
		},
		{
			name:     "3-word and phrase creates individual + combined",
			topics:   []string{"Search and Rescue"},
			expected: []string{"#Search", "#Rescue", "#SearchAndRescue"},
		},
		{
			name:     "3-word and phrase creates individual + combined",
			topics:   []string{"Disasters and Weather"},
			expected: []string{"#Disasters", "#Weather", "#DisastersAndWeather"},
		},
		{
			name:     "4-word phrase only creates individual tags",
			topics:   []string{"Structure Fires and Collapses"},
			expected: []string{"#Structure", "#Fires", "#Collapses"},
		},
		{
			name:     "Complex with 3-word and phrase",
			topics:   []string{"Disasters and Weather - Search and Rescue"},
			expected: []string{"#Disasters", "#Weather", "#DisastersAndWeather", "#Search", "#Rescue", "#SearchAndRescue"},
		},
		{
			name:     "Multiple topics with multi-word",
			topics:   []string{"Fire", "Emergency Response", "Transportation"},
			expected: []string{"#Fire", "#Emergency", "#Response", "#Transportation"},
		},
		{
			name:     "Empty topics",
			topics:   []string{},
			expected: nil,
		},
		{
			name:     "Empty string in topics",
			topics:   []string{"Fire", "", "Water"},
			expected: []string{"#Fire", "#Water"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTopicTags(tt.topics)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDeduplicateTags(t *testing.T) {
	tests := []struct {
		name     string
		tags     []string
		expected []string
	}{
		{
			name:     "No duplicates",
			tags:     []string{"#Fire", "#Water", "#Air"},
			expected: []string{"#Fire", "#Water", "#Air"},
		},
		{
			name:     "Exact duplicates",
			tags:     []string{"#Fire", "#Water", "#Fire"},
			expected: []string{"#Fire", "#Water"},
		},
		{
			name:     "Case insensitive duplicates",
			tags:     []string{"#Fire", "#FIRE", "#fire"},
			expected: []string{"#Fire"},
		},
		{
			name:     "Preserves order",
			tags:     []string{"#Zebra", "#Apple", "#Mango"},
			expected: []string{"#Zebra", "#Apple", "#Mango"},
		},
		{
			name:     "Empty list",
			tags:     []string{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deduplicateTags(tt.tags)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatHashtagText(t *testing.T) {
	tests := []struct {
		name     string
		tags     []string
		expected string
	}{
		{
			name:     "Single tag",
			tags:     []string{"#Fire"},
			expected: "🏷️ #Fire",
		},
		{
			name:     "Multiple tags",
			tags:     []string{"#Fire", "#Water", "#Air"},
			expected: "🏷️ #Fire, #Water, #Air",
		},
		{
			name:     "Empty list",
			tags:     []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatHashtagText(tt.tags)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCamelCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Single word", "fire", "Fire"},
		{"Two words", "fire truck", "FireTruck"},
		{"Multiple words", "search and rescue", "SearchAndRescue"},
		{"Already camel case", "FireTruck", "FireTruck"},
		{"Extra spaces", "fire   truck", "FireTruck"},
		{"Empty string", "", ""},
		{"Single letter", "a", "A"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := camelCase(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Benchmark tests
func BenchmarkGenerate(b *testing.B) {
	alert := backend.Alert{
		AlertType: "Flash",
		Location: &backend.Location{
			Address: "Kyiv, Ukraine",
		},
		Topics: []string{
			"Conflicts - Air",
			"Conflicts - Land",
			"Transportation - Aviation",
			"Disasters and Weather - Search and Rescue",
		},
		Headline:  "Test Alert",
		EventTime: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Generate(alert)
	}
}
