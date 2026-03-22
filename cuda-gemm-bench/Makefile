# CUDA GEMM Benchmark — Makefile
# 用法:
#   make all        编译全部
#   make run        运行全部 benchmark
#   make clean      清理

NVCC := nvcc
NVCC_FLAGS := -O3 -arch=sm_80    # sm_80 = A100, 改 sm_89 for RTX 4050
CUBLAS_LIBS := -lcublas

# 矩阵大小（可通过环境变量覆盖）
M ?= 4096
N ?= 4096
K ?= 4096
WARMUP ?= 5
ITERS ?= 20

TARGETS := naive_gemm tiled_gemm cublas_gemm

.PHONY: all run clean bench_naive bench_tiled bench_cublas

all: $(TARGETS)

naive_gemm: naive_gemm.cu
	$(NVCC) $(NVCC_FLAGS) -o $@ $<

tiled_gemm: tiled_gemm.cu
	$(NVCC) $(NVCC_FLAGS) -o $@ $<

cublas_gemm: cublas_gemm.cu
	$(NVCC) $(NVCC_FLAGS) -o $@ $< $(CUBLAS_LIBS)

# 单独运行
bench_naive: naive_gemm
	./naive_gemm $(M) $(N) $(K) $(WARMUP) $(ITERS)

bench_tiled: tiled_gemm
	./tiled_gemm $(M) $(N) $(K) $(WARMUP) $(ITERS)

bench_cublas: cublas_gemm
	./cublas_gemm $(M) $(N) $(K) $(WARMUP) $(ITERS)

# 运行全部 + 生成报告
run: all
	python3 benchmark.py --M $(M) --N $(N) --K $(K) --warmup $(WARMUP) --iters $(ITERS)

clean:
	rm -f $(TARGETS) results.json report.md
