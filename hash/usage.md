# package hash
Under the hood, hashes are stored as fixed size [64]byte.
* Enough room for bigger hashes like SHA-512
* Fixed size allows us to avoid heap allocations
* Hashes are NOT stored as [64]byte on disk (unless they really are that big)
* Recompilation (simply changing MaxHashBytes to 32) is a possibility if you're very concerned

## HashWindow
This is an under-the-hood type that adorns the [64]byte with a small fixed-size amount of metadata.
* A HashWindow can represent a hash
* A HashWindow can represent an n-byte prefix of a hash
* A HashWindow can represent an m-byte suffix of a hash

However, the caller must use the various wrappers to use these features:

## Wrapper: HashHolder
Create a HashWindow (on the stack), and call AsHashHolder() on it

You can create a 32 byte hash and load it from disk:
```go
import (
    "io"
    "os"
)

file, _ := os.Open("MyFileOfHashes")
defer file.Close()
myHashWindow := hash.HashWindow{}
hash := myHashWindow.AsHashHolder(32)
_ = hash.Read(file)

file2, _ = os.Create("MyOutputFile")
defer file2.Close()
hash.Write(file2)
```

## Wrappers: PrefixHolder, SuffixHolder
You can split the hash in a HashHolder into a PrefixHolder and a SuffixHolder. You will need to provide two more HashWindows
on the stack for this, a slight faff but it does keep us free from heap allocations.
```go
hw2 := hash.HashWindow{}
hw3 := hash.HashWindow{}
prefix, suffix := hash.SplitPrefixSuffix(&hw2, &hw3, 5)
```
You will end up with a prefix of the first 5 nibbles, and a suffix of the final 59 nibbles. You can read/write prefixes
and suffixes from disk too.

## See the code for further details, don't be shy!