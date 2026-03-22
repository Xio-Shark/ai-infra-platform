// tiled_gemm.cu — Shared Memory 分块优化 CUDA GEMM
// 核心优化：将 A 和 B 的子块加载到 Shared Memory，减少 Global Memory 访问
// 理论加速比：~TILE_SIZE 倍（每个 Global Mem 读取被重用 TILE_SIZE 次）
// 预期性能：~30 TFLOPS（A100）

#include <cuda_fp16.h>
#include <cuda_runtime.h>
#include <stdio.h>
#include <stdlib.h>
#include <sys/time.h>


#define CUDA_CHECK(call)                                                       \
  do {                                                                         \
    cudaError_t err = call;                                                    \
    if (err != cudaSuccess) {                                                  \
      fprintf(stderr, "CUDA error at %s:%d: %s\n", __FILE__, __LINE__,         \
              cudaGetErrorString(err));                                        \
      exit(EXIT_FAILURE);                                                      \
    }                                                                          \
  } while (0)

// 分块大小——每个 block 处理 TILE×TILE 的输出子矩阵
// A100 每个 SM 有 164KB shared memory，32×32×2（半精度）= 2KB per tile
// 两个 tile (A_tile + B_tile) = 4KB，远小于限制
#define TILE_SIZE 32

// ============================================================
// Tiled GEMM Kernel: C = A × B
// 通过 Shared Memory 分块减少 Global Memory 访问
// ============================================================
__global__ void tiled_gemm_kernel(const half *A, const half *B, half *C, int M,
                                  int N, int K) {
  // Shared Memory 缓冲区——存储当前 tile 的 A 和 B 子块
  __shared__ float As[TILE_SIZE][TILE_SIZE];
  __shared__ float Bs[TILE_SIZE][TILE_SIZE];

  int row = blockIdx.y * TILE_SIZE + threadIdx.y;
  int col = blockIdx.x * TILE_SIZE + threadIdx.x;

  float sum = 0.0f;

  // 沿 K 维度滑动 tile
  int num_tiles = (K + TILE_SIZE - 1) / TILE_SIZE;

  for (int t = 0; t < num_tiles; t++) {
    // 协作加载：每个线程加载 A_tile 和 B_tile 各一个元素
    int a_col = t * TILE_SIZE + threadIdx.x;
    int b_row = t * TILE_SIZE + threadIdx.y;

    // 边界检查 + 加载到 Shared Memory
    if (row < M && a_col < K) {
      As[threadIdx.y][threadIdx.x] = __half2float(A[row * K + a_col]);
    } else {
      As[threadIdx.y][threadIdx.x] = 0.0f;
    }

    if (b_row < K && col < N) {
      Bs[threadIdx.y][threadIdx.x] = __half2float(B[b_row * N + col]);
    } else {
      Bs[threadIdx.y][threadIdx.x] = 0.0f;
    }

    // 同步：确保 tile 完全加载后再计算
    __syncthreads();

    // 计算当前 tile 的部分内积
    for (int k = 0; k < TILE_SIZE; k++) {
      sum += As[threadIdx.y][k] * Bs[k][threadIdx.x];
    }

    // 同步：确保计算完成后再加载下一个 tile
    __syncthreads();
  }

  // 写回结果
  if (row < M && col < N) {
    C[row * N + col] = __float2half(sum);
  }
}

// ============================================================
// 辅助函数（与 naive_gemm.cu 相同）
// ============================================================
void init_matrix(half *mat, int rows, int cols) {
  for (int i = 0; i < rows * cols; i++) {
    mat[i] = __float2half((float)(rand() % 100) / 100.0f);
  }
}

double get_time_ms() {
  struct timeval tv;
  gettimeofday(&tv, NULL);
  return tv.tv_sec * 1000.0 + tv.tv_usec / 1000.0;
}

int main(int argc, char **argv) {
  int M = (argc > 1) ? atoi(argv[1]) : 4096;
  int N = (argc > 2) ? atoi(argv[2]) : 4096;
  int K = (argc > 3) ? atoi(argv[3]) : 4096;
  int warmup = (argc > 4) ? atoi(argv[4]) : 5;
  int iters = (argc > 5) ? atoi(argv[5]) : 20;

  printf("=== Tiled CUDA GEMM (TILE=%d) ===\n", TILE_SIZE);
  printf("矩阵大小: M=%d, N=%d, K=%d (FP16)\n", M, N, K);
  printf("Warmup: %d, Benchmark: %d 次迭代\n", warmup, iters);

  size_t size_A = M * K * sizeof(half);
  size_t size_B = K * N * sizeof(half);
  size_t size_C = M * N * sizeof(half);

  half *h_A = (half *)malloc(size_A);
  half *h_B = (half *)malloc(size_B);
  half *h_C = (half *)malloc(size_C);

  srand(42);
  init_matrix(h_A, M, K);
  init_matrix(h_B, K, N);

  half *d_A, *d_B, *d_C;
  CUDA_CHECK(cudaMalloc(&d_A, size_A));
  CUDA_CHECK(cudaMalloc(&d_B, size_B));
  CUDA_CHECK(cudaMalloc(&d_C, size_C));

  CUDA_CHECK(cudaMemcpy(d_A, h_A, size_A, cudaMemcpyHostToDevice));
  CUDA_CHECK(cudaMemcpy(d_B, h_B, size_B, cudaMemcpyHostToDevice));

  // 使用 TILE_SIZE × TILE_SIZE 的线程块
  dim3 blockDim(TILE_SIZE, TILE_SIZE);
  dim3 gridDim((N + TILE_SIZE - 1) / TILE_SIZE,
               (M + TILE_SIZE - 1) / TILE_SIZE);

  printf("Grid: (%d, %d), Block: (%d, %d)\n", gridDim.x, gridDim.y, blockDim.x,
         blockDim.y);
  printf("Shared Memory per block: %lu bytes\n",
         2 * TILE_SIZE * TILE_SIZE * sizeof(float));

  // Warmup
  for (int i = 0; i < warmup; i++) {
    tiled_gemm_kernel<<<gridDim, blockDim>>>(d_A, d_B, d_C, M, N, K);
  }
  CUDA_CHECK(cudaDeviceSynchronize());

  // Benchmark
  double start = get_time_ms();
  for (int i = 0; i < iters; i++) {
    tiled_gemm_kernel<<<gridDim, blockDim>>>(d_A, d_B, d_C, M, N, K);
  }
  CUDA_CHECK(cudaDeviceSynchronize());
  double end = get_time_ms();

  double avg_ms = (end - start) / iters;
  double flops = 2.0 * M * N * K;
  double tflops = (flops / (avg_ms / 1000.0)) / 1e12;

  printf("\n--- 结果 ---\n");
  printf("平均耗时: %.2f ms\n", avg_ms);
  printf("性能: %.2f TFLOPS\n", tflops);
  printf("效率: %.1f%% (A100 FP16 理论峰值 312 TFLOPS)\n",
         tflops / 312.0 * 100.0);

  printf("\n{\"kernel\": \"tiled\", \"tile_size\": %d, \"M\": %d, \"N\": %d, "
         "\"K\": %d, \"avg_ms\": %.4f, \"tflops\": %.4f}\n",
         TILE_SIZE, M, N, K, avg_ms, tflops);

  CUDA_CHECK(cudaFree(d_A));
  CUDA_CHECK(cudaFree(d_B));
  CUDA_CHECK(cudaFree(d_C));
  free(h_A);
  free(h_B);
  free(h_C);

  return 0;
}
