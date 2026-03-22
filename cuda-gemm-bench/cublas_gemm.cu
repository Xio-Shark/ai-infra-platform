// cublas_gemm.cu — cuBLAS 参考实现
// 调用 NVIDIA 官方 cuBLAS 库的 cublasHgemm（FP16 Tensor Core）
// 这是生产级性能上限参考，用于对比手写 kernel 的差距
// 预期性能：~250 TFLOPS（A100 FP16 Tensor Core）

#include <cublas_v2.h>
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

#define CUBLAS_CHECK(call)                                                     \
  do {                                                                         \
    cublasStatus_t status = call;                                              \
    if (status != CUBLAS_STATUS_SUCCESS) {                                     \
      fprintf(stderr, "cuBLAS error at %s:%d: %d\n", __FILE__, __LINE__,       \
              status);                                                         \
      exit(EXIT_FAILURE);                                                      \
    }                                                                          \
  } while (0)

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

  printf("=== cuBLAS GEMM (FP16 Tensor Core) ===\n");
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

  // 创建 cuBLAS handle
  cublasHandle_t handle;
  CUBLAS_CHECK(cublasCreate(&handle));

  // 启用 Tensor Core（FP16 → FP16 with FP32 accumulate）
  CUBLAS_CHECK(cublasSetMathMode(handle, CUBLAS_TENSOR_OP_MATH));

  // cuBLAS 使用 column-major，所以计算 C^T = B^T × A^T
  // 即 cublasHgemm(N, N, ..., B, N, A, K, ..., C, N)
  const half alpha_h = __float2half(1.0f);
  const half beta_h = __float2half(0.0f);

  // Warmup
  for (int i = 0; i < warmup; i++) {
    CUBLAS_CHECK(cublasHgemm(handle, CUBLAS_OP_N, CUBLAS_OP_N, N, M, K,
                             &alpha_h, d_B,
                             N,      // B 作为 "A" (column-major trick)
                             d_A, K, // A 作为 "B"
                             &beta_h, d_C, N)); // C
  }
  CUDA_CHECK(cudaDeviceSynchronize());

  // Benchmark
  double start = get_time_ms();
  for (int i = 0; i < iters; i++) {
    CUBLAS_CHECK(cublasHgemm(handle, CUBLAS_OP_N, CUBLAS_OP_N, N, M, K,
                             &alpha_h, d_B, N, d_A, K, &beta_h, d_C, N));
  }
  CUDA_CHECK(cudaDeviceSynchronize());
  double end = get_time_ms();

  double avg_ms = (end - start) / iters;
  double flops = 2.0 * M * N * K;
  double tflops = (flops / (avg_ms / 1000.0)) / 1e12;

  printf("\n--- 结果 ---\n");
  printf("平均耗时: %.2f ms\n", avg_ms);
  printf("性能: %.2f TFLOPS\n", tflops);
  printf("效率: %.1f%% (A100 FP16 Tensor Core 理论峰值 312 TFLOPS)\n",
         tflops / 312.0 * 100.0);

  printf("\n{\"kernel\": \"cublas\", \"M\": %d, \"N\": %d, \"K\": %d, "
         "\"avg_ms\": %.4f, \"tflops\": %.4f}\n",
         M, N, K, avg_ms, tflops);

  CUBLAS_CHECK(cublasDestroy(handle));
  CUDA_CHECK(cudaFree(d_A));
  CUDA_CHECK(cudaFree(d_B));
  CUDA_CHECK(cudaFree(d_C));
  free(h_A);
  free(h_B);
  free(h_C);

  return 0;
}
