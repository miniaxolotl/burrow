package protocol

import "fmt"
import "math/rand"

var idAdjectives = []string{
	"arcane", "ancient", "astral", "bold", "brave", "chaotic", "cryptic", "dark", "elder",
	"ethereal", "fierce", "frozen", "hidden", "icy", "jade", "keen", "liquid", "mystic",
	"noble", "obscure", "potent", "quick", "radiant", "shadow", "swift", "twilight",
	"uncanny", "vivid", "wandering", "wild",
}

var idNouns = []string{
	"amulet", "basilisk", "cipher", "dragon", "ember", "fortress", "gargoyle", "helm",
	"illusion", "kraken", "lich", "mithril", "nymph", "oracle", "phoenix", "quest",
	"rune", "specter", "talisman", "umbral", "void", "wyrm", "zephyr", "amethyst",
	"bramble", "crypt", "druid", "forge", "grimoire", "haven", "isle", "knave",
	"lava", "moon", "nexus", "obsidian", "prism", "quill", "shadow", "tome", "umbra",
	"vestige", "warden", "xorn", "zinc",
}

// RandomTunnelID returns a random human-readable tunnel ID.
func RandomTunnelID() string {
	adj := idAdjectives[rand.Intn(len(idAdjectives))]
	noun := idNouns[rand.Intn(len(idNouns))]
	return fmt.Sprintf("%s-%s-%d", adj, noun, rand.Intn(100))
}
