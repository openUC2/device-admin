package diskusage

import (
	"syscall"

	"github.com/pkg/errors"
)

type Usage struct {
	Size uint64
	Free uint64
}

func (u Usage) Used() uint64 {
	return u.Size - u.Free
}

func (u Usage) RatioUsed() float32 {
	return float32(u.Used()) / float32(u.Size)
}

func GetUsage(volumePath string) (u Usage, err error) {
	var result syscall.Statfs_t
	if err = syscall.Statfs(volumePath, &result); err != nil {
		return Usage{}, errors.Wrapf(err, "couldn't stat %s", volumePath)
	}

	if result.Bsize < 0 {
		return Usage{}, errors.Wrapf(err, "filesystem reports negative block size %d", result.Bsize)
	}
	u.Size = result.Blocks * uint64(result.Bsize)
	u.Free = result.Bfree * uint64(result.Bsize)
	return u, nil
}
