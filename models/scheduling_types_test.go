package models_test

import (
	"code.cloudfoundry.org/bbs/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CellState GPU matching", func() {
	var cell models.CellState

	BeforeEach(func() {
		cell = models.NewCellState(
			"cell-1", 0, "http://cell-1", models.RootFSProviders{},
			models.Resources{MemoryMB: 1024, DiskMB: 1024, Containers: 10},
			models.Resources{MemoryMB: 1024, DiskMB: 1024, Containers: 10},
			nil, nil, "z1", 0, false, nil, nil, nil, 0,
		)
		cell.GPUCapacity = models.GPUCapacity{Total: 2, Free: 2, Type: "nvidia"}
	})

	DescribeTable("ResourceMatch with a GPU request",
		func(res models.Resource, expectProblems []string) {
			err := cell.ResourceMatch(&res)
			if len(expectProblems) == 0 {
				Expect(err).NotTo(HaveOccurred())
				return
			}
			Expect(err).To(HaveOccurred())
			matchErr, ok := err.(models.InsufficientResourcesError)
			Expect(ok).To(BeTrue())
			for _, problem := range expectProblems {
				Expect(matchErr.Problems).To(HaveKey(problem))
			}
		},
		Entry("no GPU requested, always matches on GPU", models.Resource{MemoryMB: 10, DiskMB: 10}, []string(nil)),
		Entry("enough free GPUs, matching type", models.Resource{MemoryMB: 10, DiskMB: 10, GPULimit: 2, GPUType: "nvidia"}, []string(nil)),
		Entry("enough free GPUs, empty requested type matches any", models.Resource{MemoryMB: 10, DiskMB: 10, GPULimit: 1, GPUType: ""}, []string(nil)),
		Entry("not enough free GPUs", models.Resource{MemoryMB: 10, DiskMB: 10, GPULimit: 3, GPUType: "nvidia"}, []string{"gpu"}),
		Entry("type mismatch", models.Resource{MemoryMB: 10, DiskMB: 10, GPULimit: 1, GPUType: "amd"}, []string{"gpu_type"}),
	)

	It("still rejects on memory/disk/containers exactly as before, independent of GPU", func() {
		err := cell.ResourceMatch(&models.Resource{MemoryMB: 99999, DiskMB: 10})
		Expect(err).To(HaveOccurred())
		matchErr := err.(models.InsufficientResourcesError)
		Expect(matchErr.Problems).To(HaveKey("memory"))
		Expect(matchErr.Problems).NotTo(HaveKey("gpu"))
	})
})
