package versioning

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	semverRe = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)$`)
	vnumRe   = regexp.MustCompile(`^v(\d+)$`)
)

func Supported(v string) bool {
	v = strings.TrimSpace(v)
	return semverRe.MatchString(v) || vnumRe.MatchString(v)
}

func MaxSemver(versions []string) string {
	var best string
	for _, v := range versions {
		if !semverRe.MatchString(v) {
			continue
		}
		if best == "" || Compare(v, best) > 0 {
			best = v
		}
	}
	return best
}

func MaxVNumber(versions []string) int {
	best := 0
	for _, v := range versions {
		m := vnumRe.FindStringSubmatch(v)
		if len(m) != 2 {
			continue
		}
		n, _ := strconv.Atoi(m[1])
		if n > best {
			best = n
		}
	}
	return best
}

func NextPatch(v string) string {
	m := semverRe.FindStringSubmatch(v)
	if len(m) == 4 {
		maj, _ := strconv.Atoi(m[1])
		min, _ := strconv.Atoi(m[2])
		patch, _ := strconv.Atoi(m[3])
		return fmt.Sprintf("%d.%d.%d", maj, min, patch+1)
	}
	m = vnumRe.FindStringSubmatch(v)
	if len(m) == 2 {
		n, _ := strconv.Atoi(m[1])
		return fmt.Sprintf("v%d", n+1)
	}
	return "0.0.1"
}

func Compare(a, b string) int {
	am := semverRe.FindStringSubmatch(a)
	bm := semverRe.FindStringSubmatch(b)
	if len(am) == 4 && len(bm) == 4 {
		for i := 1; i <= 3; i++ {
			ai, _ := strconv.Atoi(am[i])
			bi, _ := strconv.Atoi(bm[i])
			if ai > bi {
				return 1
			}
			if ai < bi {
				return -1
			}
		}
		return 0
	}
	av := vnumRe.FindStringSubmatch(a)
	bv := vnumRe.FindStringSubmatch(b)
	if len(av) == 2 && len(bv) == 2 {
		ai, _ := strconv.Atoi(av[1])
		bi, _ := strconv.Atoi(bv[1])
		if ai > bi {
			return 1
		}
		if ai < bi {
			return -1
		}
		return 0
	}
	return strings.Compare(a, b)
}

func Greater(a, b string) bool { return Compare(a, b) > 0 }

func SortDesc(versions []string) {
	sort.Slice(versions, func(i, j int) bool { return Compare(versions[i], versions[j]) > 0 })
}
