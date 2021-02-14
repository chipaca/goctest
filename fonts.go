package main

// © 2021 John Lenton
// MIT licensed.
// from https://github.com/chipaca/goctest

type font struct {
	numerals [10][]string
	percent  []string
	passed   []string
	tests    []string
	run      []string
	space    string
}

var fonts = struct {
	braille, future, boring, double font
}{
	// based on the 'smbraille' toilet font by Sam Hocevar <sam@hocevar.net>
	// BUT, added a % sign, and made the numerals be text figures.
	braille: font{
		numerals: [10][]string{
			{" ⡔⣢", " ⠫⠜"},
			{" ⢴ ", " ⠼⠄"},
			{" ⠔⢢", " ⠮⠤"},
			{" ⠒⢲", " ⣈⡱"},
			{" ⢀⢴", " ⠉⢹"},
			{" ⡖⠒", " ⣉⡱"},
			{" ⣎⡁", " ⠣⠜"},
			{" ⠒⢲", " ⢰⠁"},
			{" ⢎⡱", " ⠣⠜"},
			{" ⡔⢢", " ⢈⡹"},
		},
		percent: []string{" ⠶⡜", " ⡜⠶"},
		passed:  []string{" ⣀⡀ ⢀⣀ ⢀⣀ ⢀⣀ ⢀⡀ ⢀⣸  ", " ⡧⠜ ⠣⠼ ⠭⠕ ⠭⠕ ⠣⠭ ⠣⠼ ⠶"},
		tests:   []string{" ⣰⡀ ⢀⡀ ⢀⣀ ⣰⡀ ⢀⣀", " ⠘⠤ ⠣⠭ ⠭⠕ ⠘⠤ ⠭⠕"},
		run:     []string{" ⡀⣀ ⡀⢀ ⣀⡀  ", " ⠏  ⠣⠼ ⠇⠸ ⠶"},
		space:   "  ",
	},

	//  based on the 'future' toilet font by Sam Hocevar <sam@hocevar.net>
	future: font{
		numerals: [10][]string{
			{"┏━┓", "┃┃┃", "┗━┛"},
			{"╺┓ ", " ┃ ", "╺┻╸"},
			{"┏━┓", "┏━┛", "┗━╸"},
			{"┏━┓", "╺━┫", "┗━┛"},
			{"╻ ╻", "┗━┫", "  ╹"},
			{"┏━╸", "┗━┓", "┗━┛"},
			{"┏━┓", "┣━┓", "┗━┛"},
			{"┏━┓", "  ┃", "  ╹"},
			{"┏━┓", "┣━┫", "┗━┛"},
			{"┏━┓", "┗━┫", "┗━┛"},
		},
		percent: []string{"┏┓╻", "┏━┛", "╹┗┛"},
		passed:  []string{"┏━┓┏━┓┏━┓┏━┓┏━╸╺┳┓ ", "┣━┛┣━┫┗━┓┗━┓┣╸  ┃┃ ", "╹  ╹ ╹┗━┛┗━┛┗━╸╺┻┛╹"},
		tests:   []string{"╺┳╸┏━╸┏━┓╺┳╸┏━┓", " ┃ ┣╸ ┗━┓ ┃ ┗━┓", " ╹ ┗━╸┗━┛ ╹ ┗━┛"},
		run:     []string{"┏━┓╻ ╻┏┓╻ ", "┣┳┛┃ ┃┃┗┫ ", "╹┗╸┗━┛╹ ╹╹"},
		space:   "  ",
	},

	// boring old not-really-a-font, mostly for testing (or boring people)
	boring: font{
		numerals: [10][]string{{"0"}, {"1"}, {"2"}, {"3"}, {"4"}, {"5"}, {"6"}, {"7"}, {"8"}, {"9"}},
		percent:  []string{"%"},
		passed:   []string{"passed."},
		tests:    []string{"tests"},
		run:      []string{"run."},
		space:    " ",
	},

	// double-width characters. Used ‘unifonter’ for this one.
	double: font{
		numerals: [10][]string{{"０"}, {"１"}, {"２"}, {"３"}, {"４"}, {"５"}, {"６"}, {"７"}, {"８"}, {"９"}},
		percent:  []string{"％"},
		passed:   []string{"ｐａｓｓｅｄ．"},
		tests:    []string{"ｔｅｓｔｓ"},
		run:      []string{"ｒｕｎ．"},
		space:    "　",
	},
}
