package git

func (r *Repo) SeedDemo() error {
	files := map[string]string{
		"README.md":     "# GoGit Demo\n\nA teaching repository that stores objects like Git.\n\n- Blob / Tree / Commit\n- Staging index\n- Branches and three-way merge\n",
		"src/hello.txt": "hello\n",
		"docs/guide.md": "# Guide\n\nStart on `main`, then inspect objects in the archive viewer.\n",
		".gogitignore":  "*.tmp\n*.swp\n.DS_Store\n",
	}
	for p, c := range files {
		if err := r.WriteWorktree(p, c); err != nil {
			return err
		}
	}
	if _, err := r.Add([]string{"README.md", "src", "docs", ".gogitignore"}); err != nil {
		return err
	}
	if _, err := r.Commit("Initial commit — archive sealed", "Archivist <archivist@gogit.local>"); err != nil {
		return err
	}
	if _, err := r.CreateBranch("feature/docs"); err != nil {
		return err
	}
	if err := r.Checkout("feature/docs"); err != nil {
		return err
	}
	if err := r.WriteWorktree("docs/guide.md", "# Guide\n\nStart on `main`, then inspect objects in the archive viewer.\n\n## Feature branch\nThis paragraph was added on `feature/docs`.\n"); err != nil {
		return err
	}
	if _, err := r.Add([]string{"docs/guide.md"}); err != nil {
		return err
	}
	if _, err := r.Commit("docs: expand guide on feature branch", "Archivist <archivist@gogit.local>"); err != nil {
		return err
	}
	return r.Checkout("main")
}
