package protocol

import (
	"crypto/rand"
	"math/big"
	"strings"
)

var idAdjectives = []string{
	"ancient", "arcane", "astral", "bold", "brave", "chaotic", "cryptic", "dark", "elder",
	"ethereal", "fierce", "frozen", "hidden", "icy", "jade", "keen", "liquid", "mighty",
	"mystic", "noble", "obscure", "potent", "quick", "radiant", "silent", "swift",
	"twilight", "uncanny", "vivid", "wandering", "wild",
}

var idNouns = []string{
	"amulet", "basilisk", "cipher", "dragon", "ember", "fortress", "gargoyle", "helm",
	"illusion", "kraken", "lich", "mithril", "nymph", "oracle", "phoenix", "rune",
	"specter", "talisman", "void", "wyrm", "zephyr", "amethyst", "bramble", "crypt",
	"druid", "forge", "grimoire", "haven", "isle", "lava", "moon", "nexus", "obsidian",
	"prism", "quill", "tome", "vestige", "warden", "xorn",
}

func randIndex(n int) int {
	max := big.NewInt(int64(n))
	i, err := rand.Int(rand.Reader, max)
	if err != nil {
		panic(err)
	}
	return int(i.Int64())
}

// RandomTunnelID returns a random three-word tunnel ID, e.g. "mighty-arcane-dragon".
func RandomTunnelID() string {
	parts := []string{
		idAdjectives[randIndex(len(idAdjectives))],
		idAdjectives[randIndex(len(idAdjectives))],
		idNouns[randIndex(len(idNouns))],
	}
	return strings.Join(parts, "-")
}
