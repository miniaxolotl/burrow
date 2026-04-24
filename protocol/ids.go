package protocol

import (
	"crypto/rand"
	"math/big"
	"strings"
)

var idAdverbs = []string{
	"arcaneily", "blindly", "boldly", "brightly", "calmly", "chaotically", "clearly",
	"coldly", "covertly", "cruelly", "cryptically", "darkly", "dauntlessly", "deeply",
	"deftly", "dimly", "distantly", "divinely", "dreadfully", "dryly", "eerily",
	"eldritchly", "endlessly", "eternally", "evilly", "faintly", "fearlessly", "fiercely",
	"firmly", "forebodingly", "freely", "frostily", "fully", "ghostly", "ghoulishly",
	"gravelely", "grimly", "hauntingly", "harshly", "heavily", "hellishly", "hollowly",
	"icily", "infernally", "keenly", "lethally", "lightly", "liminally", "lowly",
	"magically", "malevolently", "menacingly", "mercilessly", "mutely", "mystically",
	"nimbly", "nobly", "obscurely", "ominously", "openly", "perilously", "phantomly",
	"proudly", "quietly", "rapidly", "rarely", "relentlessly", "roughly", "ruinously",
	"savagely", "sharply", "silently", "sinisterly", "slowly", "softly", "solemnly",
	"solidly", "spectrally", "starkly", "stealthily", "sternly", "stolidly", "strongly",
	"subtly", "swiftly", "terribly", "thinly", "treacherously", "truly", "undyingly",
	"unholy", "vastly", "vengefully", "vividly", "voraciously", "wickedly", "wildly",
	"wisely", "wrathfully", "wryly",
}

var idAdjectives = []string{
	"abyssal", "accursed", "ancient", "arcane", "ashen", "astral", "banished", "battered",
	"bewitched", "bleak", "blighted", "bloodied", "bold", "bonded", "brave", "broken",
	"burning", "celestial", "chaotic", "charmed", "chromatic", "cold", "corrupted",
	"crimson", "cryptic", "cursed", "dark", "dead", "deathly", "defiled", "demonic",
	"destined", "diabolical", "distant", "divine", "doomed", "draconic", "dread",
	"druidic", "dry", "dwarven", "dying", "elder", "eldritch", "elven", "empty",
	"enchanted", "ethereal", "exalted", "fallen", "feral", "fierce", "fiendish",
	"flaming", "forbidden", "forgotten", "forsaken", "foul", "frozen", "furtive",
	"ghostly", "gilded", "glowing", "grim", "hallowed", "haunted", "hellish", "heretical",
	"hidden", "hollow", "holy", "hungry", "icy", "infernal", "iron", "jade", "keen",
	"legendary", "lethal", "liquid", "lost", "luminous", "lurking", "mad", "malevolent",
	"mighty", "molten", "moonlit", "mournful", "murky", "mystic", "necrotic", "noble",
	"obscure", "ominous", "pale", "petrified", "phantom", "plagued", "potent", "primal",
	"profane", "quick", "radiant", "raging", "ruined", "runic", "sacred", "savage",
	"scarlet", "scorched", "sepulchral", "shadow", "shattered", "silent", "silver",
	"sinister", "skeletal", "smoldering", "spectral", "stark", "still", "stone",
	"storming", "sunken", "swift", "tainted", "terrible", "twilight", "twisted",
	"umbral", "uncanny", "unholy", "unseen", "veiled", "vengeful", "vivid", "volatile",
	"wandering", "wicked", "wild", "withered", "wrathful", "wretched",
}

var idNouns = []string{
	"altar", "amulet", "anvil", "arch", "archmage", "artefact", "assassin", "axe",
	"banshee", "basilisk", "beacon", "behemoth", "blade", "blight", "bones", "bramble",
	"catacomb", "centaur", "chains", "chimera", "cipher", "citadel", "crypt", "curse",
	"cyclops", "dagger", "demon", "depths", "dirge", "dragon", "druid", "dungeon",
	"effigy", "ember", "enchantment", "exile", "familiar", "fiend", "forge", "fortress",
	"gargoyle", "gate", "ghost", "ghoul", "giant", "goblin", "golem", "grave",
	"grimoire", "guardian", "harbinger", "haven", "helm", "heretic", "hex", "hydra",
	"idol", "illusion", "inferno", "isle", "jailer", "kraken", "labyrinth", "lair",
	"lance", "leviathan", "lich", "longbow", "manticore", "mausoleum", "maze", "minotaur",
	"mithril", "monolith", "moon", "necromancer", "nexus", "nightmare", "nymph",
	"obsidian", "ogre", "oracle", "orc", "overlord", "paladin", "phantom", "phoenix",
	"plague", "portal", "prism", "prophet", "quill", "ravine", "reaper", "relic",
	"revenant", "rune", "sanctum", "sarcophagus", "scroll", "sentinel", "serpent",
	"shade", "shard", "shrine", "siege", "skeleton", "skull", "specter", "spell",
	"spire", "staff", "stalker", "stronghold", "sword", "talisman", "throne", "tomb",
	"tome", "tower", "troll", "unicorn", "urn", "vampire", "vault", "vestige",
	"void", "vortex", "warden", "warlock", "wasteland", "witch", "wizard", "wraith",
	"wyvern", "xorn", "zealot", "zephyr", "zombie",
}

func randIndex(n int) int {
	max := big.NewInt(int64(n))
	i, err := rand.Int(rand.Reader, max)
	if err != nil {
		panic(err)
	}
	return int(i.Int64())
}

// RandomTunnelID returns a random four-word tunnel ID, e.g. "swiftly-ancient-silent-dragon".
func RandomTunnelID() string {
	parts := []string{
		idAdverbs[randIndex(len(idAdverbs))],
		idAdjectives[randIndex(len(idAdjectives))],
		idAdjectives[randIndex(len(idAdjectives))],
		idNouns[randIndex(len(idNouns))],
	}
	return strings.Join(parts, "-")
}
