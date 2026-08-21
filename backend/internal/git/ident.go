package git

import (
	"fmt"
	"strings"
)

func formatIdent(author string, unix int64) string {
	name, email := splitAuthor(author)
	return fmt.Sprintf("%s <%s> %d +0800", name, email, unix)
}

func splitAuthor(author string) (string, string) {
	author = strings.TrimSpace(author)
	if i := strings.Index(author, "<"); i >= 0 {
		j := strings.Index(author, ">")
		if j > i {
			name := strings.TrimSpace(author[:i])
			email := strings.TrimSpace(author[i+1 : j])
			if name == "" {
				name = "GoGit"
			}
			if email == "" {
				email = "gogit@local"
			}
			return name, email
		}
	}
	if author == "" {
		author = "GoGit"
	}
	return author, "gogit@local"
}

func parseIdent(val string) (string, int64) {
	lt := strings.LastIndex(val, ">")
	if lt < 0 {
		return strings.TrimSpace(val), 0
	}
	ident := strings.TrimSpace(val[:lt+1])
	rest := strings.TrimSpace(val[lt+1:])
	fields := strings.Fields(rest)
	var unix int64
	if len(fields) > 0 {
		fmt.Sscanf(fields[0], "%d", &unix)
	}
	return ident, unix
}

func FormatAuthor(name, email string) string {
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)
	if name == "" {
		name = "GoGit"
	}
	if email == "" {
		email = "gogit@local"
	}
	return fmt.Sprintf("%s <%s>", name, email)
}
