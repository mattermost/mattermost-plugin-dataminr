// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package hashtag

import "strings"

// extractTopicTags extracts hashtags from alert topics.
//
// Algorithm:
// 1. Split each topic on " - " delimiter (category/subcategory separator)
// 2. For each segment:
//   - Create hashtags for every word except "and"
//   - If the segment has exactly 3 words with "and" in the middle,
//     also create a combined CamelCase hashtag
//
// Examples:
//   - "Conflicts - Air" -> #Conflicts, #Air
//   - "Disasters and Weather - Search and Rescue" -> #Disasters, #Weather, #DisastersAndWeather, #Search, #Rescue, #SearchAndRescue
//   - "Structure Fires and Collapses" -> #Structure, #Fires, #Collapses
//   - "Oil and Gas" -> #Oil, #Gas, #OilAndGas
//
// 3. Return all tags (deduplication happens later)
func extractTopicTags(topics []string) []string {
	if len(topics) == 0 {
		return nil
	}

	var allTags []string

	for _, topic := range topics {
		if topic == "" {
			continue
		}

		// Split on " - " delimiter (category - subcategory)
		segments := strings.SplitSeq(topic, " - ")

		for segment := range segments {
			segment = strings.TrimSpace(segment)
			if segment == "" {
				continue
			}

			// Split into words
			words := strings.Fields(segment)
			if len(words) == 0 {
				continue
			}

			// Create hashtag for every word except "and"
			for _, word := range words {
				if strings.ToLower(word) != "and" {
					allTags = append(allTags, "#"+capitalizeFirst(word))
				}
			}

			// If exactly 3 words with "and" in the middle, also create combined hashtag
			if len(words) == 3 && strings.ToLower(words[1]) == "and" {
				combined := capitalizeFirst(words[0]) + "And" + capitalizeFirst(words[2])
				allTags = append(allTags, "#"+combined)
			}
		}
	}

	return allTags
}

// capitalizeFirst capitalizes the first letter of a word.
func capitalizeFirst(word string) string {
	if len(word) == 0 {
		return ""
	}
	return strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
}
