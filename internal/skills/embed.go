package skills

import "embed"

// Files embeds the teach skill — the single skill shipped with the pharos
// binary. It carries the teaching philosophy (mission, lesson, learning
// record, zone of proximal development, storage strength) plus the pharos
// CLI reference (references/pharos-cli.md) as a context pointer, so one
// install delivers both the teaching guidance and the command reference.
//
//go:embed teach
var Files embed.FS

// All is the list of embedded skills installed by `pharos skills install`.
var All = []string{"teach"}

// SkillName is the primary (and only) shipped skill.
const SkillName = "teach"
