package auth

import "strings"

var (
	FORMS = []string{
		"Очная",
		"Очно-заочная",
		"Заочная",
	}

	EDU_LEVEL = []string{
		"Бакалавриат",
		"Магистратура",
		"Аспирантура",
	}

	PAYMENTS = []string{
		"Контракт",
		"Бюджет",
		"Целевое",
	}
)

// TODO: replace with regexp
func isValidSnils(s string) bool {
	if len(s) == 14 {
		for i := range s {
			switch i {
			case 3, 7:
				if s[i] != '-' {
					return false
				}
			case 11:
				if s[i] != ' ' {
					return false
				}
			default:
				if !strings.Contains("0123456789", string(s[i])) {
					return false
				}
			}
		}
		return true
	}
	return false
}

func isValidForm(s string) bool {
	for _, v := range FORMS {
		if v == s {
			return true
		}
	}
	return false
}

func isValidPayment(s string) bool {
	for _, v := range PAYMENTS {
		if v == s {
			return true
		}
	}
	return false
}

func isValidEduLevel(s string) bool {
	for _, v := range EDU_LEVEL {
		if v == s {
			return true
		}
	}
	return false
}
