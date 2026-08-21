package git

import "testing"

func TestFsckHealthyRepo(t *testing.T) {
	r := initRepo(t, SHA1)
	_ = r.WriteWorktree("f.txt", "ok\n")
	_, _ = r.Add([]string{"f.txt"})
	_, err := r.Commit("c", "T <t@t>")
	if err != nil {
		t.Fatal(err)
	}
	rep, err := r.Fsck()
	if err != nil {
		t.Fatal(err)
	}
	if !rep.OK {
		t.Fatalf("issues %+v", rep.Issues)
	}
	if rep.ObjectCount == 0 || rep.Reachable == 0 {
		t.Fatalf("counts %+v", rep)
	}
}

func TestConfigRoundtrip(t *testing.T) {
	r := initRepo(t, SHA1)
	cfg, err := r.Config()
	if err != nil {
		t.Fatal(err)
	}
	cfg.UserName = "Ada"
	cfg.UserEmail = "ada@gogit.local"
	if err := r.SetConfig(cfg); err != nil {
		t.Fatal(err)
	}
	got, err := r.Config()
	if err != nil {
		t.Fatal(err)
	}
	if got.UserName != "Ada" || got.Author() != "Ada <ada@gogit.local>" {
		t.Fatalf("%+v", got)
	}
	got.HashAlgo = SHA256
	if err := r.SetConfig(got); err == nil {
		t.Fatal("algo change should fail")
	}
}
