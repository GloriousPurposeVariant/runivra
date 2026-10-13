package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	optNoEnterprise = "No, Community only"
	optToken        = "Yes, clone it with a Git token"
	optCopy         = "Yes, copy it from a local folder (not recommended)"
	optStart        = "Start setup"
	optCancel       = "Cancel"
	optInstall      = "Yes, install Docker"
	optNoInstall    = "No, continue without it"
	optStartNow     = "Yes, start it"
	optStartLater   = "No, I will start it later"
	optStackDocker  = "Everything in Docker"
	optStackHostDB  = "Odoo in Docker, PostgreSQL on this computer"
	optInstallPG    = "Yes, install PostgreSQL"
	optNoInstallPG  = "No, cancel this choice"
	optExistingUser = "Use a database user that already exists"
	optCreateUser   = "Create a new database user"
)

const tokenHelp = "Typing is hidden. The token stays on this computer: it is not saved to any file and is only handed to Git for this download."

const firstOptionRow = 7

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	askStyle    = lipgloss.NewStyle().Bold(true)
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	activeStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	doneStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
)

type Answers struct {
	Environment     string
	Version         string
	Path            string
	Name            string
	EnterpriseToken string
	EnterprisePath  string
	CustomRepo      string
	CustomBranch    string
	CustomToken     string
	Port            int
	InstallDocker   bool
	Start           bool
	HostDatabase    bool
	InstallPostgres bool
	DBUser          string
	DBPassword      string
	CreateDBUser    bool
	DBAdminPassword string
}

type Options struct {
	DockerMissing     bool
	PostgresInstalled bool

	// NeedsAdminPassword is true where creating a database user takes the
	// postgres administrator's password, which is the case on Windows.
	NeedsAdminPassword bool
	// CheckLogin and CheckAdmin test a login while the user is still on the
	// question. Either may be nil, which skips the test.
	CheckLogin func(user string, password string) error
	CheckAdmin func(password string) error
	// RoleExists reports whether a database user name is already taken. It
	// may be nil where that cannot be found out without prompting.
	RoleExists func(adminPassword string, name string) (bool, error)
	PortIsFree func(port int) bool
	Versions   []string
}

type question struct {
	key     string
	title   string
	help    string
	options []string
	secret  bool
	summary bool
	show    func(answers map[string]string) bool
	browse  bool
	initial string
	check   func(input string) (string, bool)
	// verify runs when the answer is accepted and can look at earlier
	// answers. Unlike check it may be slow, so it does not run per keystroke.
	verify func(answers map[string]string) (string, bool)
}

func portCheck(isFree func(port int) bool) func(input string) (string, bool) {
	return func(input string) (string, bool) {
		port, err := strconv.Atoi(strings.TrimSpace(input))
		if err != nil || port < 1 || port > 65535 {
			return "Type a port number between 1 and 65535.", false
		}
		if isFree != nil && !isFree(port) {
			return fmt.Sprintf("Port %d is already in use. Try another one.", port), false
		}
		return fmt.Sprintf("Port %d is free.", port), true
	}
}

// userCheck refuses names Odoo cannot run as.
func userCheck(input string) (string, bool) {
	switch strings.TrimSpace(input) {
	case "":
		return "Type a user name.", false
	case "postgres":
		return "Odoo refuses to run as postgres. Give another user.", false
	}
	return "", true
}

func requiredCheck(input string) (string, bool) {
	if strings.TrimSpace(input) == "" {
		return "A password is needed.", false
	}
	return "", true
}

func questions(options Options) []question {
	wantsToken := func(a map[string]string) bool { return a["enterprise"] == optToken }
	wantsCopy := func(a map[string]string) bool { return a["enterprise"] == optCopy }
	hasRepo := func(a map[string]string) bool { return a["customRepo"] != "" }
	isDev := func(a map[string]string) bool { return a["environment"] == "development" }
	noDocker := func(a map[string]string) bool { return options.DockerMissing }
	willHaveDocker := func(a map[string]string) bool {
		return isDev(a) && (!options.DockerMissing || a["docker"] == optInstall)
	}

	hostDB := func(a map[string]string) bool { return isDev(a) && a["stack"] == optStackHostDB }
	askInstallPG := func(a map[string]string) bool { return hostDB(a) && !options.PostgresInstalled }
	askMode := func(a map[string]string) bool { return hostDB(a) && options.PostgresInstalled }
	existingUser := func(a map[string]string) bool { return askMode(a) && a["dbMode"] != optCreateUser }
	newUser := func(a map[string]string) bool {
		return hostDB(a) && (a["postgres"] == optInstallPG || (askMode(a) && a["dbMode"] == optCreateUser))
	}
	askAdmin := func(a map[string]string) bool {
		return askMode(a) && a["dbMode"] == optCreateUser && options.NeedsAdminPassword
	}

	verifyLogin := func(a map[string]string) (string, bool) {
		if options.CheckLogin == nil {
			return "", true
		}
		if err := options.CheckLogin(a["dbUser"], a["dbPassword"]); err != nil {
			return err.Error(), false
		}
		return "", true
	}
	verifyNewUser := func(a map[string]string) (string, bool) {
		if options.RoleExists == nil {
			return "", true
		}
		exists, err := options.RoleExists(a["dbAdmin"], a["dbUser"])
		if err != nil {
			return err.Error(), false
		}
		if exists {
			return "A database user named " + a["dbUser"] + " already exists. Choose another name, or go back and use the existing user.", false
		}
		return "", true
	}
	verifyAdmin := func(a map[string]string) (string, bool) {
		if options.CheckAdmin == nil {
			return "", true
		}
		if err := options.CheckAdmin(a["dbAdmin"]); err != nil {
			return err.Error(), false
		}
		return "", true
	}
	versions := options.Versions
	if len(versions) == 0 {
		versions = []string{"20.0", "19.0", "18.0", "17.0", "16.0"}
	}

	return []question{
		{key: "environment", title: "Which environment are you setting up?", options: []string{"development", "staging", "production"}},
		{key: "version", title: "Which Odoo version?", options: versions},
		{key: "stack", title: "Where should the database run?", help: "Odoo itself runs in Docker either way.", options: []string{optStackDocker, optStackHostDB}, show: isDev},
		{key: "path", title: "Where should the project live?", browse: true},
		{key: "name", title: "Name for a new project folder", help: "Created inside that location. Leave empty to build directly in it; it must then be empty."},
		{key: "enterprise", title: "Do you need Odoo Enterprise?", options: []string{optNoEnterprise, optToken, optCopy}},
		{key: "enterpriseToken", title: "Paste your Git token for Odoo Enterprise", help: tokenHelp, secret: true, show: wantsToken},
		{key: "enterprisePath", title: "Where is your local copy of Odoo Enterprise?", help: "Type the full path of the folder.", show: wantsCopy},
		{key: "customRepo", title: "Git repository of your custom addons", help: "Paste the repository address, or leave empty to start with an empty custom folder."},
		{key: "customBranch", title: "Which branch do you develop on?", help: "Leave empty to use the Odoo version as the branch name.", show: hasRepo},
		{key: "customToken", title: "Git token for that repository", help: "Leave empty for a public repository. " + tokenHelp, secret: true, show: hasRepo},
		{key: "postgres", title: "PostgreSQL was not found on this computer. Install it now?", help: "The stack you chose needs PostgreSQL installed here.", options: []string{optInstallPG, optNoInstallPG}, show: askInstallPG},
		{key: "dbMode", title: "Which database user should Odoo use?", help: "PostgreSQL is already installed on this computer.", options: []string{optExistingUser, optCreateUser}, show: askMode},
		{key: "dbUser", title: "Existing database user", help: "A PostgreSQL user that is allowed to create databases.", initial: "odoo", check: userCheck, show: existingUser},
		{key: "dbPassword", title: "Password of that user", help: "Typing is hidden. It is checked now and only written to this project's odoo.conf.", secret: true, check: requiredCheck, verify: verifyLogin, show: existingUser},
		{key: "dbAdmin", title: "PostgreSQL administrator password", help: "The password of the postgres user, set when PostgreSQL was installed. Needed once to create the user. Typing is hidden and it is not saved.", secret: true, check: requiredCheck, verify: verifyAdmin, show: askAdmin},
		{key: "dbUser", title: "Name for the new database user", help: "Runivra will create this user for Odoo.", initial: "odoo", check: userCheck, verify: verifyNewUser, show: newUser},
		{key: "dbPassword", title: "Password for the new user", help: "Choose a password, or leave empty to have one generated. Typing is hidden.", secret: true, show: newUser},
		{key: "port", title: "Which port should Odoo use on this computer?", help: "You will open Odoo at http://localhost:<port>.", initial: "8069", check: portCheck(options.PortIsFree), show: isDev},
		{key: "docker", title: "Docker was not found. Install it now?", help: "Docker and Docker Compose are needed to run the project.", options: []string{optInstall, optNoInstall}, show: noDocker},
		{key: "start", title: "Start Odoo when setup finishes?", options: []string{optStartNow, optStartLater}, show: willHaveDocker},
		{key: "confirm", title: "Ready to start?", options: []string{optStart, optCancel}, summary: true},
	}
}

type model struct {
	questions []question
	answers   map[string]string
	index     int
	cursor    int
	input     string
	finished  bool
	browser   browser
	note      string
	noteOK    bool
	height    int
	width     int
	offset    int
}

func newModel() model {
	return newModelWith(Options{})
}

func newModelWith(options Options) model {
	return model{questions: questions(options), answers: map[string]string{}}
}

func (m *model) validate() {
	m.note, m.noteOK = "", true
	if check := m.current().check; check != nil {
		m.note, m.noteOK = check(m.input)
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) current() question {
	return m.questions[m.index]
}

func (m model) visible(index int) bool {
	show := m.questions[index].show
	return show == nil || show(m.answers)
}

func (m *model) load() {
	q := m.current()
	m.input = m.answers[q.key]
	if m.input == "" {
		m.input = q.initial
	}
	m.validate()

	m.cursor = 0
	if q.browse {
		m.open(m.startFolder())
		return
	}
	for index, option := range q.options {
		if option == m.answers[q.key] {
			m.cursor = index
		}
	}
}

func (m *model) forward() {
	for m.index < len(m.questions)-1 {
		m.index++
		if m.visible(m.index) {
			break
		}
	}
	m.load()
}

func (m *model) back() bool {
	for index := m.index - 1; index >= 0; index-- {
		if m.visible(index) {
			m.index = index
			m.load()
			return true
		}
	}
	return false
}

func (m *model) accept() bool {
	q := m.current()
	switch {
	case q.browse:
		if !m.pick() {
			return false
		}
	case len(q.options) > 0:
		m.answers[q.key] = q.options[m.cursor]
	default:
		if !m.noteOK {
			return false
		}
		m.answers[q.key] = strings.TrimSpace(m.input)
		if q.verify != nil {
			if message, ok := q.verify(m.answers); !ok {
				m.note, m.noteOK = message, false
				return false
			}
		}

	}

	if q.key == "confirm" {
		m.finished = m.answers[q.key] == optStart
		return true
	}
	m.forward()
	return false
}

func (m model) summaryLines() []string {
	a := m.answers
	folder := a["path"]
	if folder == "" {
		folder = "the current folder"
	}
	if a["name"] != "" {
		folder += " / " + a["name"]
	}
	custom := "empty folder"
	if a["customRepo"] != "" {
		custom = a["customRepo"]
	}
	database := "in Docker"
	if a["stack"] == optStackHostDB {
		database = "PostgreSQL on this computer, user " + a["dbUser"]
		if a["postgres"] == optInstallPG {
			database += " (PostgreSQL will be installed)"
		}
		if a["dbMode"] == optCreateUser {
			database += " (will be created)"
		}

	}
	return []string{
		"Environment    " + a["environment"],
		"Odoo version   " + a["version"],
		"Folder         " + folder,
		"Enterprise     " + a["enterprise"],
		"Custom addons  " + custom,
		"Database       " + database,
	}

}

// wrap breaks text into lines no wider than width, at spaces where it can.
// A width of zero or less means "unknown", and the text is left alone.
func wrap(text string, width int) []string {
	if width <= 0 || len([]rune(text)) <= width {
		return []string{text}
	}
	var lines []string
	line := ""
	for _, word := range strings.Fields(text) {
		for len([]rune(word)) > width {
			if line != "" {
				lines = append(lines, line)
				line = ""
			}
			runes := []rune(word)
			lines = append(lines, string(runes[:width]))
			word = string(runes[width:])
		}
		switch {
		case line == "":
			line = word
		case len([]rune(line))+1+len([]rune(word)) <= width:
			line += " " + word
		default:
			lines = append(lines, line)
			line = word
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

// textWidth is the room for text inside the wizard's left and right margins.
func (m model) textWidth() int {
	if m.width == 0 {
		return 0
	}
	return max(m.width-4, 20)
}

// helpLines is the help text of the current question, wrapped to the window.
func (m model) helpLines() []string {
	help := m.current().help
	if m.current().browse {
		help = m.browser.dir
		if help == "" {
			help = "This computer"
		}
	}
	return wrap(help, m.textWidth())
}

func (m model) optionStart() int {
	start := firstOptionRow + len(m.helpLines()) - 1
	if m.current().summary {
		start += len(m.summaryLines()) + 1
	}
	return start
}

func (m model) capacity() int {
	total := len(m.options())
	if m.height == 0 {
		return total
	}
	reserved := m.optionStart() + 3
	if m.current().browse {
		reserved += 2
	}
	room := m.height - reserved
	if room < 3 {
		room = 3
	}
	if room > total {
		room = total
	}
	return room
}

func (m *model) scroll() {
	room := m.capacity()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+room {
		m.offset = m.cursor - room + 1
	}
	if limit := len(m.options()) - room; m.offset > limit {
		m.offset = limit
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	q := m.current()
	options := m.options()
	isChoice := len(options) > 0

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
	case tea.KeyMsg:

		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEsc:
			if !m.back() {
				return m, tea.Quit
			}
		case tea.KeyEnter:
			if m.accept() {
				return m, tea.Quit
			}
		case tea.KeyUp:
			if isChoice && m.cursor > 0 {
				m.cursor--
			}
		case tea.KeyDown:
			if isChoice && m.cursor < len(options)-1 {
				m.cursor++
			}
		case tea.KeyBackspace:
			if q.browse {
				m.search(trimLast(m.browser.filter))
			} else if !isChoice {
				m.input = trimLast(m.input)
				m.validate()
			}
		case tea.KeyRunes, tea.KeySpace:
			if q.browse {
				m.search(m.browser.filter + string(msg.Runes))
			} else if !isChoice {
				m.input += string(msg.Runes)
				m.validate()
			}
		}
	case tea.MouseMsg:
		if isChoice && msg.Button == tea.MouseButtonWheelUp && m.cursor > 0 {
			m.cursor--
			break
		}
		if isChoice && msg.Button == tea.MouseButtonWheelDown && m.cursor < len(options)-1 {
			m.cursor++
			break
		}
		row := msg.Y - m.optionStart()
		if !isChoice || row < 0 || row >= m.capacity() {
			break
		}
		m.cursor = m.offset + row

		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			if m.accept() {
				return m, tea.Quit
			}
		}
	}
	m.scroll()
	return m, nil

}

func (m model) progress() string {
	total, position := 0, 0
	for index := range m.questions {
		if !m.visible(index) {
			continue
		}
		total++
		if index <= m.index {
			position++
		}
	}
	dots := doneStyle.Render(strings.Repeat("●", position)) + dimStyle.Render(strings.Repeat("○", total-position))
	return dots + dimStyle.Render(fmt.Sprintf("  Step %d of %d", position, total))
}

func (m model) View() string {
	q := m.current()
	options := m.options()
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString("  " + titleStyle.Render("Runivra setup") + "\n")
	b.WriteString("  " + m.progress() + "\n")
	b.WriteString("\n")
	b.WriteString("  " + askStyle.Render(q.title) + "\n")
	for _, line := range m.helpLines() {
		b.WriteString("  " + dimStyle.Render(line) + "\n")
	}
	b.WriteString("\n")

	if q.summary {
		for _, line := range m.summaryLines() {
			b.WriteString("  " + line + "\n")
		}
		b.WriteString("\n")
	}

	if len(options) > 0 {
		end := m.offset + m.capacity()
		for index := m.offset; index < end; index++ {
			option := options[index]

			if index == m.cursor {
				b.WriteString("  " + activeStyle.Render("❯ "+option) + "\n")
			} else {
				b.WriteString("    " + option + "\n")
			}
		}
		more := ""
		if hidden := len(options) - end; hidden > 0 || m.offset > 0 {
			more = fmt.Sprintf("%d above · %d below · ", m.offset, hidden)
		}
		if q.browse {
			b.WriteString("\n  " + dimStyle.Render("Search: ") + m.browser.filter + activeStyle.Render("█") + "\n")
			b.WriteString("\n  " + dimStyle.Render(more+"Type to search · ↑/↓ or mouse · Enter or click to open · Esc to go back"))
		} else {
			b.WriteString("\n  " + dimStyle.Render(more+"↑/↓ or mouse to choose · Enter or click to confirm · Esc to go back"))
		}
	} else {
		shown := m.input
		if q.secret {
			shown = strings.Repeat("•", len([]rune(m.input)))
		}
		if m.note != "" && m.noteOK {
			for _, line := range wrap("✔ "+m.note, m.textWidth()) {
				b.WriteString("\n  " + doneStyle.Render(line))
			}
			b.WriteString("\n")
		} else if m.note != "" {
			for _, line := range wrap("✘ "+m.note, m.textWidth()) {
				b.WriteString("\n  " + errorStyle.Render(line))
			}
			b.WriteString("\n")
		}

		b.WriteString("  " + activeStyle.Render("❯ ") + shown + activeStyle.Render("█") + "\n")
		b.WriteString("\n  " + dimStyle.Render("Type your answer · Enter to continue · Esc to go back"))
	}
	b.WriteString("\n")
	return b.String()
}

func Run(options Options) (Answers, bool, error) {
	program := tea.NewProgram(newModelWith(options), tea.WithAltScreen(), tea.WithMouseAllMotion())

	final, err := program.Run()
	if err != nil {
		return Answers{}, false, err
	}
	m := final.(model)
	if !m.finished {
		return Answers{}, false, nil
	}
	a := m.answers
	port, _ := strconv.Atoi(a["port"])

	return Answers{
		Environment:     a["environment"],
		Version:         a["version"],
		Path:            a["path"],
		Name:            a["name"],
		EnterpriseToken: a["enterpriseToken"],
		EnterprisePath:  a["enterprisePath"],
		CustomRepo:      a["customRepo"],
		CustomBranch:    a["customBranch"],
		CustomToken:     a["customToken"],
		Port:            port,
		InstallDocker:   a["docker"] == optInstall,
		Start:           a["start"] == optStartNow,
		HostDatabase:    a["stack"] == optStackHostDB,
		InstallPostgres: a["postgres"] == optInstallPG,
		DBUser:          a["dbUser"],
		DBPassword:      a["dbPassword"],
		CreateDBUser:    a["dbMode"] == optCreateUser,
		DBAdminPassword: a["dbAdmin"],
	}, true, nil
}
