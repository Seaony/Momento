package lock

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/Seaony/Momento/app/internal/i18n"
	"golang.org/x/crypto/argon2"
)

var (
	ErrPasswordTooShort = i18n.NewError("error_password_short")
	ErrInvalidHash      = i18n.NewError("error_password_hash")
	ErrWrongPassword    = i18n.NewError("error_password_wrong")
	ErrCooldown         = i18n.NewError("error_password_cooldown")
)

const (
	memory     = 64 * 1024
	iterations = 3
	threads    = 4
	saltLength = 16
	keyLength  = 32
	cooldown   = 30 * time.Second
)

func Hash(password string) (string, error) {
	if !utf8.ValidString(password) || utf8.RuneCountInString(password) < 4 {
		return "", ErrPasswordTooShort
	}
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, iterations, memory, threads, keyLength)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, memory, iterations, threads, base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key)), nil
}

func Verify(encoded, password string) (bool, error) {
	if len(encoded) > 256 {
		return false, ErrInvalidHash
	}
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" || parts[2] != "v=19" {
		return false, ErrInvalidHash
	}
	cost := strings.Split(parts[3], ",")
	if len(cost) != 3 {
		return false, ErrInvalidHash
	}
	params := make([]uint64, 3)
	for i, prefix := range []string{"m=", "t=", "p="} {
		if !strings.HasPrefix(cost[i], prefix) {
			return false, ErrInvalidHash
		}
		value, err := strconv.ParseUint(strings.TrimPrefix(cost[i], prefix), 10, 32)
		if err != nil {
			return false, ErrInvalidHash
		}
		params[i] = value
	}
	// 配置文件可被改坏，先限制消耗再调用密码算法。
	if params[0] < memory || params[0] > 256*1024 || params[1] < 1 || params[1] > 6 || params[2] < 1 || params[2] > 8 {
		return false, ErrInvalidHash
	}
	salt, err := base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if err != nil || len(salt) != saltLength {
		return false, ErrInvalidHash
	}
	key, err := base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if err != nil || len(key) != keyLength {
		return false, ErrInvalidHash
	}
	actual := argon2.IDKey([]byte(password), salt, uint32(params[1]), uint32(params[0]), uint8(params[2]), keyLength)
	return subtle.ConstantTimeCompare(actual, key) == 1, nil
}

type Guard struct {
	mu           sync.Mutex
	failures     int
	blockedUntil time.Time
	now          func() time.Time
}

func NewGuard() *Guard { return &Guard{now: time.Now} }

func (g *Guard) Check(encoded, password string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	now := g.time()
	if now.Before(g.blockedUntil) {
		return ErrCooldown
	}
	if !g.blockedUntil.IsZero() {
		g.failures, g.blockedUntil = 0, time.Time{}
	}
	valid, err := Verify(encoded, password)
	if err != nil {
		return err
	}
	if valid {
		g.failures = 0
		return nil
	}
	g.failures++
	if g.failures >= 5 {
		g.blockedUntil = g.time().Add(cooldown)
		return ErrCooldown
	}
	return ErrWrongPassword
}

func (g *Guard) Remaining() time.Duration {
	g.mu.Lock()
	defer g.mu.Unlock()
	return max(0, g.blockedUntil.Sub(g.time()))
}

func (g *Guard) Reset() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.failures, g.blockedUntil = 0, time.Time{}
}

func (g *Guard) time() time.Time {
	if g.now != nil {
		return g.now()
	}
	return time.Now()
}
