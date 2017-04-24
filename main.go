package main

import "fmt"
import "math"
import "hash/fnv"

import (
	"github.com/GaryBoone/GoStats/stats"
	"github.com/Sirupsen/logrus"
	"github.com/golang/groupcache/consistenthash"
	"github.com/serialx/hashring"
)
import jump "github.com/renstrom/go-jump-consistent-hash"

type mappingFunc func([]string, []string) map[string]string

var keys, shards, moreshards, fewershards []string

func init() {
	numKeys := 100000
	numShards := 10
	numMoreShards := 2
	numFewerShards := 2

	logrus.Infof("Running tests for hashing. numKeys=%d, numShards=%d, numMoreShards=%d, numFewerShards=%d", numKeys, numShards, numMoreShards, numFewerShards)

	keys = make([]string, 0)
	for i := 0; i < numKeys; i++ {
		keys = append(keys, string(i))
	}
	shards = make([]string, 0)
	for i := 0; i < numShards; i++ {
		shards = append(shards, fmt.Sprintf("host%d", i))
	}
	// copy shards
	moreshards = shards[:]
	for i := 0; i < numMoreShards; i++ {
		moreshards = append(moreshards, fmt.Sprintf("host%d", numShards+i))
	}
	fewershards = shards[:(numShards - numFewerShards)]
}

// Print output differences for key
func diffOutput(msg string, a, b map[string]string) {
	match := 0
	for k, v := range a {
		if v == b[k] {
			match++
		}
	}
	logrus.Infof("%s: %v%%, std-dev %d", msg, (float64(match) / float64(len(a)) * 100), keysPerHost(b))
}

func keysPerHost(a map[string]string) int {
	counter := make(map[string]int)
	for _, v := range a {
		counter[v]++
	}

	nums := make([]float64, 0)

	for _, count := range counter {
		nums = append(nums, float64(count))
	}
	return int(stats.StatsSampleStandardDeviation(nums))

}

// Run the tests against a given mapping func
func runTests(name string, f mappingFunc) {
	var aa, a, b, c map[string]string

	// baseline
	a = f(keys, shards)
	// re-do baseline
	aa = f(keys, shards)
	// more shards
	b = f(keys, moreshards)
	// fewer shards
	c = f(keys, fewershards)
	logrus.Infof("%s", name)
	// Output stuff
	diffOutput("Same", a, aa)
	diffOutput("add", a, b)
	diffOutput("remove", a, c)
	logrus.Infof("")
}

// The goal here is to test various hashing scenarios to see what works out
// how we want
func main() {
	// mod
	runTests("Mod", Mod)

	// Consistent hashing ring
	runTests("hashRing", hashRing)

	// consistent hash ring with more replicas
	runTests("consistenthash-100replica", hashRing2)

	// jump hashing
	runTests("JumpHashing", JumpHashing)
}

// Various implementations of
// Take a list of keys, and return the mapping of key -> hostname
func Mod(keys, shards []string) map[string]string {
	mapping := make(map[string]string)
	for _, key := range keys {
		h := fnv.New32a()
		h.Write([]byte(key))
		mapping[key] = shards[int(math.Mod(float64(h.Sum32()), float64(len(shards))))]
	}
	return mapping
}

func hashRing(keys, shards []string) map[string]string {
	mapping := make(map[string]string)

	ring := hashring.New(shards)
	for _, key := range keys {
		mapping[key], _ = ring.GetNode(key)
	}
	return mapping
}

func hashRing2(keys, shards []string) map[string]string {

	mapping := make(map[string]string)

	ring := consistenthash.New(100, nil)
	for _, h := range shards {
		ring.Add(h)
	}
	for _, key := range keys {
		mapping[key] = ring.Get(key)
	}
	return mapping
}

func JumpHashing(keys, shards []string) map[string]string {
	mapping := make(map[string]string)
	for _, key := range keys {
		mapping[key] = shards[jump.HashString(key, int32(len(shards)), jump.CRC64)]
	}
	return mapping
}
