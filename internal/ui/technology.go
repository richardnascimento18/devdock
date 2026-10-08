package ui

// Technology is presentation metadata for semantic detector labels.
type Technology struct {
	Label string
	Badge string
	Color string
}

var techRegistry = map[string]Technology{
	"Go":           {Label: "Go", Badge: "🐹 Go", Color: "81"},
	"Rust":         {Label: "Rust", Badge: "🦀 Rust", Color: "208"},
	"C/C++":        {Label: "C/C++", Badge: "⚙ C/C++", Color: "250"},
	"F#":           {Label: "F#", Badge: "F#", Color: "135"},
	"Fastify":      {Label: "Fastify", Badge: "Fastify", Color: "245"},
	"C#":           {Label: "C#", Badge: "# C#", Color: "135"},
	"Java":         {Label: "Java", Badge: "☕ Java", Color: "166"},
	"Kotlin":       {Label: "Kotlin", Badge: "𝕂 Kotlin", Color: "99"},
	"Swift":        {Label: "Swift", Badge: "🐦 Swift", Color: "202"},
	"Zig":          {Label: "Zig", Badge: "⚡ Zig", Color: "214"},
	"Haskell":      {Label: "Haskell", Badge: "λ Haskell", Color: "135"},
	"Erlang":       {Label: "Erlang", Badge: "⟳ Erlang", Color: "196"},
	"Elixir":       {Label: "Elixir", Badge: "💧 Elixir", Color: "128"},
	"Clojure":      {Label: "Clojure", Badge: "( Clojure", Color: "34"},
	"Scala":        {Label: "Scala", Badge: "◆ Scala", Color: "196"},
	"OCaml":        {Label: "OCaml", Badge: "🐪 OCaml", Color: "208"},
	"Node":         {Label: "Node", Badge: "⬡ Node", Color: "34"},
	"Deno":         {Label: "Deno", Badge: "🦕 Deno", Color: "47"},
	"Bun":          {Label: "Bun", Badge: "🍞 Bun", Color: "214"},
	"Python":       {Label: "Python", Badge: "🐍 Python", Color: "226"},
	"Ruby":         {Label: "Ruby", Badge: "💎 Ruby", Color: "196"},
	"PHP":          {Label: "PHP", Badge: "🐘 PHP", Color: "105"},
	"Perl":         {Label: "Perl", Badge: "🐪 Perl", Color: "75"},
	"HTML":         {Label: "HTML", Badge: "🌐 HTML", Color: "202"},
	"CSS":          {Label: "CSS", Badge: "🎨 CSS", Color: "39"},
	"TypeScript":   {Label: "TypeScript", Badge: "Ts TS", Color: "33"},
	"JavaScript":   {Label: "JavaScript", Badge: "Js JS", Color: "227"},
	"React":        {Label: "React", Badge: "⚛ React", Color: "39"},
	"Next.js":      {Label: "Next.js", Badge: "▲ Next.js", Color: "255"},
	"Vue":          {Label: "Vue", Badge: "◀ Vue", Color: "41"},
	"Nuxt":         {Label: "Nuxt", Badge: "◀ Nuxt", Color: "41"},
	"Svelte":       {Label: "Svelte", Badge: "◆ Svelte", Color: "202"},
	"SvelteKit":    {Label: "SvelteKit", Badge: "◆ SvelteKit", Color: "202"},
	"Angular":      {Label: "Angular", Badge: "⬡ Angular", Color: "196"},
	"Astro":        {Label: "Astro", Badge: "🚀 Astro", Color: "213"},
	"Remix":        {Label: "Remix", Badge: "♻ Remix", Color: "135"},
	"Solid":        {Label: "Solid", Badge: "◆ Solid", Color: "33"},
	"Vite":         {Label: "Vite", Badge: "⚡ Vite", Color: "213"},
	"Nest.js":      {Label: "Nest.js", Badge: "🐱 Nest", Color: "196"},
	"Express":      {Label: "Express", Badge: "⚡ Express", Color: "245"},
	"Electron":     {Label: "Electron", Badge: "⚛ Electron", Color: "75"},
	"Django":       {Label: "Django", Badge: "🎸 Django", Color: "34"},
	"Flask":        {Label: "Flask", Badge: "🧪 Flask", Color: "245"},
	"FastAPI":      {Label: "FastAPI", Badge: "⚡ FastAPI", Color: "47"},
	"Rails":        {Label: "Rails", Badge: "🛤 Rails", Color: "196"},
	"Laravel":      {Label: "Laravel", Badge: "🔶 Laravel", Color: "208"},
	"Spring":       {Label: "Spring", Badge: "🍃 Spring", Color: "34"},
	"Flutter":      {Label: "Flutter", Badge: "🦋 Flutter", Color: "39"},
	"Dart":         {Label: "Dart", Badge: "🎯 Dart", Color: "39"},
	"React Native": {Label: "React Native", Badge: "⚛ RN", Color: "39"},
	"Docker":       {Label: "Docker", Badge: "🐋 Docker", Color: "39"},
	"Terraform":    {Label: "Terraform", Badge: "⬡ TF", Color: "99"},
	"Ansible":      {Label: "Ansible", Badge: "⚙ Ansible", Color: "196"},
	"Nix":          {Label: "Nix", Badge: "❄ Nix", Color: "75"},
	"Shell":        {Label: "Shell", Badge: "$ Shell", Color: "245"},
}

func Tech(label string) Technology {
	if t, ok := techRegistry[label]; ok {
		return t
	}
	return Technology{Label: label, Badge: label, Color: "245"}
}
