package lock

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestHashUsesRandomSaltAndVerifies(t *testing.T) {
	first, err := Hash("四个文字")
	if err != nil {
		t.Fatal(err)
	}
	second, err := Hash("四个文字")
	if err != nil {
		t.Fatal(err)
	}
	if first == second || strings.Contains(first, "四个文字") || !strings.HasPrefix(first, "$argon2id$v=19$m=65536,t=3,p=4$") {
		t.Fatal("没有正确存储带随机盐的参数串")
	}
	for _, password := range []string{"四个文字", "wrong", ""} {
		valid, err := Verify(first, password)
		if err != nil || valid != (password == "四个文字") {
			t.Fatalf("密码判断错误: %v, %v", valid, err)
		}
	}
	for _, password := range []string{"", "123", "三个字", "\xffabc"} {
		if _, err := Hash(password); !errors.Is(err, ErrPasswordTooShort) {
			t.Fatalf("接受了过短或非法密码: %v", err)
		}
	}
}

func TestVerifyRejectsInvalidOrUnboundedParameters(t *testing.T) {
	encoded := "$argon2id$v=19$m=65536,t=3,p=4$AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	cases := []string{"", "plaintext", strings.Repeat("a", 257), strings.Replace(encoded, "v=19", "v=20", 1), strings.Replace(encoded, "argon2id", "argon2i", 1)}
	for _, bad := range []string{"m=0,t=3,p=4", "m=4294967295,t=3,p=4", "m=65536,t=0,p=4", "m=65536,t=9999,p=4", "m=65536,t=3,p=0", "m=65536,t=3,p=256", "t=3,m=65536,p=4", "m=-1,t=3,p=4", "m=65536,t=3,p=4,extra=1"} {
		cases = append(cases, strings.Replace(encoded, "m=65536,t=3,p=4", bad, 1))
	}
	cases = append(cases, encoded+"=", strings.Replace(encoded, "$AAAAAAAAAAAAAAAAAAAAAA$", "$bad$", 1))
	for _, value := range cases {
		if valid, err := Verify(value, "test"); valid || !errors.Is(err, ErrInvalidHash) {
			t.Fatalf("接受了损坏参数串: %v", err)
		}
	}
}

func TestGuardEnforcesCooldownAndResetsOnlyAfterSuccess(t *testing.T) {
	encoded, err := Hash("test")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1000, 0)
	g := NewGuard()
	g.now = func() time.Time { return now }
	for range 4 {
		if err := g.Check(encoded, "wrong"); !errors.Is(err, ErrWrongPassword) {
			t.Fatal(err)
		}
	}
	if err := g.Check(encoded, "wrong"); !errors.Is(err, ErrCooldown) {
		t.Fatal(err)
	}
	if g.Remaining() != 30*time.Second {
		t.Fatalf("等待时间错误: %v", g.Remaining())
	}
	if err := g.Check(encoded, "test"); !errors.Is(err, ErrCooldown) {
		t.Fatal("冷却期间正确密码绕过了限制")
	}
	now = now.Add(29 * time.Second)
	if g.Remaining() != time.Second {
		t.Fatal(g.Remaining())
	}
	now = now.Add(time.Second)
	if err := g.Check(encoded, "test"); err != nil || g.Remaining() != 0 {
		t.Fatal(err)
	}
	for range 4 {
		if err := g.Check(encoded, "wrong"); !errors.Is(err, ErrWrongPassword) {
			t.Fatal(err)
		}
	}
	if err := g.Check(encoded, "test"); err != nil {
		t.Fatal(err)
	}
	for range 4 {
		if err := g.Check(encoded, "wrong"); !errors.Is(err, ErrWrongPassword) {
			t.Fatal("成功后失败次数没有清零")
		}
	}
	g.Reset()
	if err := g.Check("corrupt", "test"); !errors.Is(err, ErrInvalidHash) {
		t.Fatal("损坏配置没有阻止解锁")
	}
	if g.failures != 0 {
		t.Fatal("损坏配置算成了密码输错")
	}
}
