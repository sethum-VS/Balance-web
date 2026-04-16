package templates

import "strings"

var sfToMaterialIcon = map[string]string{
	"bolt":                "bolt",
	"bolt.fill":           "bolt",
	"book":                "menu_book",
	"book.closed":         "menu_book",
	"figure.walk":         "directions_walk",
	"briefcase":           "work",
	"briefcase.fill":      "work",
	"gamecontroller":      "sports_esports",
	"gamecontroller.fill": "sports_esports",
	"desktopcomputer":     "computer",
	"laptopcomputer":      "laptop_mac",
	"iphone":              "phone_iphone",
	"figure.yoga":         "self_improvement",
	"dumbbell":            "fitness_center",
	"heart":               "favorite",
	"heart.fill":          "favorite",
	"leaf":                "eco",
	"leaf.fill":           "eco",
	"fork.knife":          "restaurant",
	"cart":                "shopping_cart",
	"cart.fill":           "shopping_cart",
	"car":                 "directions_car",
	"car.fill":            "directions_car",
	"music.note":          "music_note",
	"film":                "movie",
	"graduationcap":       "school",
	"doc.text":            "description",
	"clock":               "schedule",
	"clock.fill":          "schedule",
	"moon":                "dark_mode",
	"moon.fill":           "dark_mode",
	"sun.max":             "light_mode",
	"sun.max.fill":        "light_mode",
	"house":               "home",
	"house.fill":          "home",
}

// TranslateIcon maps Apple SF Symbol names to Material Symbols names for web rendering.
func TranslateIcon(symbol string) string {
	key := strings.ToLower(strings.TrimSpace(symbol))
	if key == "" {
		return "category"
	}

	if v, ok := sfToMaterialIcon[key]; ok {
		return v
	}

	if strings.HasSuffix(key, ".fill") {
		if v, ok := sfToMaterialIcon[strings.TrimSuffix(key, ".fill")]; ok {
			return v
		}
	}

	return "category"
}
