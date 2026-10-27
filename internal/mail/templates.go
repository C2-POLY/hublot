// SPDX-License-Identifier: AGPL-3.0-or-later

package mail

import (
	"fmt"
	"strings"
)

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

// statusWords holds, per portal status, the subject and the one sentence of the
// status e-mail; %s is the request title.
var statusWords = map[string]map[string][2]string{
	"fr": {
		"received":    {"Votre demande est bien reçue", "Votre demande « %s » est bien reçue."},
		"planned":     {"Votre demande est planifiée", "Votre demande « %s » est planifiée."},
		"in_progress": {"Votre demande est en réalisation", "L'équipe travaille en ce moment sur votre demande « %s »."},
		"in_test":     {"Votre demande est en test", "Votre demande « %s » est prête, l'équipe la vérifie avant de la mettre en ligne."},
		"available":   {"Votre demande est disponible", "Votre demande « %s » est maintenant disponible."},
		"answered":    {"Votre question a sa réponse", "L'équipe a répondu à votre question « %s »."},
		"closed":      {"Votre demande est fermée", "Votre demande « %s » est fermée."},
	},
	"en": {
		"received":    {"Your request was received", "Your request “%s” was received."},
		"planned":     {"Your request is planned", "Your request “%s” is planned."},
		"in_progress": {"Your request is in the works", "The team is working on your request “%s” right now."},
		"in_test":     {"Your request is being tested", "Your request “%s” is ready, the team is checking it before it goes live."},
		"available":   {"Your request is available", "Your request “%s” is now available."},
		"answered":    {"Your question was answered", "The team answered your question “%s”."},
		"closed":      {"Your request is closed", "Your request “%s” is closed."},
	},
}

// StatusChanged tells a request's author that its status changed, in their language
// (French by default). It carries the title only, never the request's content. It
// reports false for a status it has no words for.
func StatusChanged(to, lang, name, title, status, link, preferencesLink string) (Message, bool) {
	if lang != "en" {
		lang = "fr"
	}
	words, ok := statusWords[lang][status]
	if !ok {
		return Message{}, false
	}
	title = strings.Join(strings.Fields(title), " ")
	sentence := fmt.Sprintf(words[1], title)
	if lang == "en" {
		hello := "Hi,"
		if name != "" {
			hello = "Hi " + name + ","
		}
		return Message{
			To:      to,
			Subject: words[0] + ": " + title,
			Text: fmt.Sprintf("%s\n\n%s\n\nSee my request:\n%s\n\n"+
				"You get this e-mail because you filed this request. Choose my notifications: %s\n",
				hello, sentence, link, preferencesLink),
		}, true
	}
	hello := "Bonjour,"
	if name != "" {
		hello = "Bonjour " + name + ","
	}
	return Message{
		To:      to,
		Subject: words[0] + " : " + title,
		Text: fmt.Sprintf("%s\n\n%s\n\nVoir ma demande :\n%s\n\n"+
			"Vous recevez cet e-mail car vous avez fait cette demande. Choisir mes notifications : %s\n",
			hello, sentence, link, preferencesLink),
	}, true
}
