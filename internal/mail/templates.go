// SPDX-License-Identifier: AGPL-3.0-or-later

package mail

import "fmt"

// MagicLink is the sign-in e-mail, in the person's language (French by default).
func MagicLink(to, lang, link string, validMinutes int) Message {
	if lang == "en" {
		return Message{
			To:      to,
			Subject: "Your sign-in link",
			Text: fmt.Sprintf("Hi,\n\nHere is your link to sign in. It works once, for %d minutes:\n\n%s\n\n"+
				"You did not ask for it? Ignore this e-mail, nothing will happen.\n", validMinutes, link),
		}
	}
	return Message{
		To:      to,
		Subject: "Votre lien pour entrer",
		Text: fmt.Sprintf("Bonjour,\n\nVoici votre lien pour entrer. Il marche une fois, pendant %d minutes :\n\n%s\n\n"+
			"Vous n'avez rien demandé ? Ignorez cet e-mail, il ne se passera rien.\n", validMinutes, link),
	}
}
