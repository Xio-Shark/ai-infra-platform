// naive_gemm.cu — 朴素 CUDA GEMM 实现
// 每个线程计算结果矩阵 C 中的一个元素
// 性能瓶颈：每次从 Global Memory 读取 A 和 B，带宽受限
// 预期性能：~2 TFLOPS（A100 理论峰值 312 TFLOPS FP16）

#include <cuda_runtime.h>
#include <cuda_fp16.h>
#include <stdio.h>
#include <stdlib.h>
#include <sys/time.h>

// 错误检查宏
#define CUDA_CHECK(call)                                                       \
    do {                                                                       \
        cudaError_t err = call;                                                \
        if (err != cudaSuccess) {                                              \
            fprintf(stderr, "CUDA error at %s:%d: %s\n", __FILE__, __LINE__,  \
                    cudaGetErrorString(err));                                   \
            exit(EXIT_FAILURE);                                                \
        }                                                                      \
    } while (0)

// ============================================================
// 朴素 GEMM Kernel: C = A × B
// A: M×K, B: K×N, C: M×N (Row-major)
// 每个线程负责 C[row][col] = sum(A[row][k] * B[k][col])
// ============================================================
__global__ void naive_gemm_kernel(const half *A, const half *B, half *C,
                                   int M, int N, int K) {
    // 当前线程负责的输出位置
    int row = blockIdx.y * blockDim.y + threadIdx.y;
    int col = blockIdx.x * blockDim.x + threadIdx.x;

    if (row < M && col < N) {
        float sum = 0.0f;
        // 遍历 K 维度，累加内积
        for (int k = 0; k < K; k++) {
            sum += __half2float(A[row * K + k]) * __half2float(B[k * N + col]);
        }
        C[row * N + col] = __float2half(sum);
    }
}

// ============================================================
// 辅助函数
// ============================================================

// 初始化矩阵（随机值）
void init_matrix(half *mat, int rows, int cols) {
    for (int i = 0; i < rows * cols; i++) {
        mat[i] = __float2half((float)(rand() % 100) / 100.0f);
    }
}

// 获取当前时间（微秒）
double get_time_ms() {
    struct timeval tv;
    gettimeofday(&tv, NULL);
    return tv.tv_sec * 1000.0 + tv.tv_usec / 1000.0;
}

// ============================================================
// Main: 运行 benchmark
// 用法: ./naive_gemm [M] [N] [K] [warmup_iters] [bench_iters]
// ============================================================
int main(int argc, char **argv) {
    // 默认参数
    int M = (argc > 1) ? atoi(argv[1]) : 4096;
    int N = (argc > 2) ? atoi(argv[2]) : 4096;
    int K = (argc > 3) ? atoi(argv[3]) : 4096;
    int warmup = (argc > 4) ? atoi(argv[4]) : 5;
    int iters  = (argc > 5) ? atoi(argv[5]) : 20;

    printf("=== Naive CUDA GEMM ===\n");
    printf("矩阵大小: M=%d, N=%d, K=%d (FP16)\n", M, N, K);
    printf("Warmup: %d, Benchmark: %d 次迭代\n", warmup, iters);

    // 分配 Host 内存
    size_t size_A = M * K * sizeof(half);
    size_t size_B = K * N * sizeof(half);
    size_t size_C = M * N * sizeof(half);

    half *h_A = (half *)malloc(size_A);
    half *h_B = (half *)malloc(size_B);
    half *h_C = (half *)malloc(size_C);

    // 初始化
    srand(42);
    init_matrix(h_A, M, K);
    init_matrix(h_B, K, N);

    // 分配 Device 内存
    half *d_A, *d_B, *d_C;
    CUDA_CHECK(cudaMalloc(&d_A, size_A));
    CUDA_CHECK(cudaMalloc(&d_B, size_B));
    CUDA_CHECK(cudaMalloc(&d_C, size_C));

    // Host → Device
    CUDA_CHECK(cudaMemcpy(d_A, h_A, size_A, cudaMemcpyHostToDevice));
    CUDA_CHECK(cudaMemcpy(d_B, h_B, size_B, cudaMemcpyHostToDevice));

    // 配置线程块: 16×16 = 256 threads per block
    dim3 blockDim(16, 16);
    dim3 gridDim((N + blockDim.x - 1) / blockDim.x,
                 (M + blockDim.y - 1) / blockDim.y);

    printf("Grid: (%d, %d), Block: (%d, %d)\n",
           gridDim.x, gridDim.y, blockDim.x, blockDim.y);

    // Warmup
    for (int i = 0; i < warmup; i++) {
        naive_gemm_kernel<<<gridDim, blockDim>>>(d_A, d_B, d_C, M, N, K);
    }
    CUDA_CHECK(cudaDeviceSynchronize());

    // Benchmark
    double start = get_time_ms();
    for (int i = 0; i < iters; i++) {
        naive_gemm_kernel<<<gridDim, blockDim>>>(d_A, d_B, d_C, M, N, K);
    }
    CUDA_CHECK(cudaDeviceSynchronize());
    double end = get_time_ms();

    // 计算性能
    double avg_ms = (end - start) / iters;
    // FLOPS = 2 * M * N * K（乘法 + 加法各一次）
    double flops = 2.0 * M * N * K;
    double tflops = (flops / (avg_ms / 1000.0)) / 1e12;

    printf("\n--- 结果 ---\n");
    printf("平均耗时: %.2f ms\n", avg_ms);
    printf("性能: %.2f TFLOPS\n", tflops);
    printf("效率: %.1f%% (A100 FP16 理论峰值 312 TFLOPS)\n",
           tflops / 312.0 * 100.0);

    // 输出 JSON（供 benchmark.py 解析）
    printf("\n{\"kernel\": \"naive\", \"M\": %d, \"N\": %d, \"K\": %d, "
           "\"avg_ms\": %.4f, \"tflops\": %.4f}\n",
           M, N, K, avg_ms, tflops);

    // 清理
    CUDA_CHECK(cudaFree(d_A));
    CUDA_CHECK(cudaFree(d_B));
    CUDA_CHECK(cudaFree(d_C));
    free(h_A);
    free(h_B);
    free(h_C);

    return 0;
}
