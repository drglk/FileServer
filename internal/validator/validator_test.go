package validator

import "testing"

func TestIsValidPassword(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name     string
		Password string
		Want     bool
	}{
		{
			Name:     "empty",
			Password: "",
			Want:     false,
		},
		{
			Name:     "1 lower",
			Password: "a",
			Want:     false,
		},
		{
			Name:     "8 lower",
			Password: "qwertyui",
			Want:     false,
		},
		{
			Name:     "1 Upper 7 lower",
			Password: "Qwertyui",
			Want:     false,
		},
		{
			Name:     "1 Upper 6 lower 1 digit",
			Password: "Qwertyu1",
			Want:     false,
		},
		{
			Name:     "1 Upper 5 lower 1 digit 1 symbol",
			Password: "Qwerty1!",
			Want:     true,
		},
	}

	for _, test := range tests {
		if res := IsValidPassword(test.Password); res != test.Want {
			t.Errorf("\ntest: %s\nvalue: %v\nexpected: %v", test.Name, res, test.Want)
		}
	}
}

func TestIsValidLogin(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name  string
		Login string
		Want  bool
	}{
		{
			Name:  "empty",
			Login: "",
			Want:  false,
		},
		{
			Name:  "1 lower",
			Login: "a",
			Want:  false,
		},
		{
			Name:  "8 lower",
			Login: "qwertyui",
			Want:  true,
		},
		{
			Name:  "1 Upper 7 lower",
			Login: "Qwertyui",
			Want:  true,
		},
		{
			Name:  "1 Upper 6 lower 1 digit",
			Login: "Qwertyu1",
			Want:  true,
		},
		{
			Name:  "1 Upper 5 lower 1 digit 1 symbol",
			Login: "Qwerty1!",
			Want:  false,
		},
	}

	for _, test := range tests {
		if res := IsValidLogin(test.Login); res != test.Want {
			t.Errorf("\ntest: %s\nvalue: %v\nexpected: %v", test.Name, res, test.Want)
		}
	}
}
