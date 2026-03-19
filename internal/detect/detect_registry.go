package detect

// Tech holds everything DevDock knows about a detected technology.
type TechStrc struct {
	Label string
	Badge string
	Color string
}

var techRegistry = map[string]TechStrc{
	"Go":           {Label: "Go", Badge: "🐹 Go", Color: "81"},
	"Rust":         {Label: "Rust", Badge: "🦀 Rust", Color: "208"},
	"C/C++":        {Label: "C/C++", Badge: "⚙ C/C++", Color: "250"},
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

func Tech(label string) TechStrc {
	if t, ok := techRegistry[label]; ok {
		return t
	}
	return TechStrc{Label: label, Badge: label, Color: "245"}
}

var primaryMarkers = []struct {
	file  string
	label string
}{
	{"go.mod", "Go"},
	{"Cargo.toml", "Rust"},
	{"pyproject.toml", "Python"},
	{"setup.py", "Python"},
	{"setup.cfg", "Python"},
	{"requirements.txt", "Python"},
	{"Pipfile", "Python"},
	{"pom.xml", "Java"},
	{"build.gradle", "Java"},
	{"build.gradle.kts", "Kotlin"},
	{".csproj", "C#"},
	{".fsproj", "C#"},
	{"composer.json", "PHP"},
	{"Gemfile", "Ruby"},
	{"mix.exs", "Elixir"},
	{"pubspec.yaml", "Dart"},
	{"CMakeLists.txt", "C/C++"},
	{"flake.nix", "Nix"},
	{"default.nix", "Nix"},
	{"deno.json", "Deno"},
	{"deno.jsonc", "Deno"},
	{"bun.lockb", "Bun"},
	{"*.cabal", "Haskell"},
	{"stack.yaml", "Haskell"},
	{"rebar.config", "Erlang"},
	{"mix.lock", "Elixir"},
	{"project.clj", "Clojure"},
	{"build.sbt", "Scala"},
	{"*.opam", "OCaml"},
	{"dune-project", "OCaml"},
	{"Package.swift", "Swift"},
	{".zig", "Zig"},
	{"build.zig", "Zig"},
}

var secondaryMarkers = []struct {
	file  string
	label string
}{
	{"next.config.js", "Next.js"},
	{"next.config.ts", "Next.js"},
	{"next.config.mjs", "Next.js"},
	{"nuxt.config.js", "Nuxt"},
	{"nuxt.config.ts", "Nuxt"},
	{"svelte.config.js", "SvelteKit"},
	{"svelte.config.ts", "SvelteKit"},
	{"astro.config.mjs", "Astro"},
	{"astro.config.ts", "Astro"},
	{"remix.config.js", "Remix"},
	{"vite.config.js", "Vite"},
	{"vite.config.ts", "Vite"},
	{"vite.config.mjs", "Vite"},
	{"angular.json", "Angular"},
	{"solid.config.ts", "Solid"},
	{"electron-builder.yml", "Electron"},
	{"electron-builder.json", "Electron"},
	{"manage.py", "Django"},
	{"wsgi.py", "Django"},
	{"asgi.py", "Django"},
	{"Dockerfile", "Docker"},
	{"docker-compose.yml", "Docker"},
	{"docker-compose.yaml", "Docker"},
	{"*.tf", "Terraform"},
	{"*.tfvars", "Terraform"},
	{"ansible.cfg", "Ansible"},
	{"playbook.yml", "Ansible"},
}

var directoryMarkers = []struct {
	dir   string
	label string
}{
	{"node_modules", "Node"},
	{".next", "Next.js"},
	{".nuxt", "Nuxt"},
	{".svelte-kit", "SvelteKit"},
	{".astro", "Astro"},
	{".angular", "Angular"},
	{"vendor", ""},
	{"__pycache__", "Python"},
	{".venv", "Python"},
	{"venv", "Python"},
	{"env", "Python"},
	{"target", ""},
	{".gradle", "Java"},
	{"ios", "Swift"},
	{"android", ""},
	{".terraform", "Terraform"},
	{".ansible", "Ansible"},
}

var extensionMap = map[string]string{
	".go":     "Go",
	".rs":     "Rust",
	".py":     "Python",
	".rb":     "Ruby",
	".php":    "PHP",
	".java":   "Java",
	".kt":     "Kotlin",
	".swift":  "Swift",
	".cs":     "C#",
	".fs":     "C#",
	".cpp":    "C/C++",
	".c":      "C/C++",
	".h":      "C/C++",
	".hpp":    "C/C++",
	".ts":     "TypeScript",
	".tsx":    "TypeScript",
	".js":     "JavaScript",
	".jsx":    "JavaScript",
	".vue":    "Vue",
	".svelte": "Svelte",
	".html":   "HTML",
	".htm":    "HTML",
	".css":    "CSS",
	".scss":   "CSS",
	".sass":   "CSS",
	".less":   "CSS",
	".ex":     "Elixir",
	".exs":    "Elixir",
	".erl":    "Erlang",
	".hs":     "Haskell",
	".clj":    "Clojure",
	".scala":  "Scala",
	".ml":     "OCaml",
	".dart":   "Dart",
	".zig":    "Zig",
	".sh":     "Shell",
	".bash":   "Shell",
	".zsh":    "Shell",
	".fish":   "Shell",
	".pl":     "Perl",
	".pm":     "Perl",
	".tf":     "Terraform",
}

var frameworkDeps = []struct {
	dep   string
	label string
}{
	{"next", "Next.js"},
	{"nuxt", "Nuxt"},
	{"@nuxtjs/composition-api", "Nuxt"},
	{"svelte", "Svelte"},
	{"@sveltejs/kit", "SvelteKit"},
	{"astro", "Astro"},
	{"@remix-run/node", "Remix"},
	{"@remix-run/react", "Remix"},
	{"solid-js", "Solid"},
	{"@solidjs/start", "SvelteKit"},
	{"@angular/core", "Angular"},
	{"vue", "Vue"},
	{"react-native", "React Native"},
	{"expo", "React Native"},
	{"react", "React"},
	{"electron", "Electron"},
	{"express", "Express"},
	{"fastify", "Express"},
	{"@nestjs/core", "Nest.js"},
	{"vite", "Vite"},
	{"typescript", "TypeScript"},
	{"django", "Django"},
	{"flask", "Flask"},
	{"fastapi", "FastAPI"},
}

var pythonFrameworkMarkers = []struct {
	name  string
	label string
}{
	{"django", "Django"},
	{"flask", "Flask"},
	{"fastapi", "FastAPI"},
	{"tornado", "Python"},
	{"aiohttp", "Python"},
	{"starlette", "Python"},
}
