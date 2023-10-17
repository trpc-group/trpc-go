English | [中文](benchmark.zh_CN.md)

# Benchmark

## Environment

* Tencent Cloud Standard SA2 CVM, equipped with an AMD EPYC™ Rome processor
  * uname -a
    * Linux VM-0-13-centos 3.10.0-1160.92.1.el7.x86_64 #1 SMP Tue Jun 20 11:48:01 UTC 2023 x86_64 x86_64 x86_64 GNU/Linux
  * cat /proc/cpuinfo
    * Refer to the appendix (8 cores, 2.60GHz)
* Memory: 16GB
* Network environment: The average ping latency between the calling and called machines is around 0.2ms
* Go version used during compilation: 1.21.2
* Server-side: Bound to 8 cores (taskset -c 0-7)
  * Logic: Using the trpc protocol to echo the string "hello"
    * Corresponding packet length for sending: 122
    * Corresponding packet length for receiving: 27
* Client-side: Bound to 8 cores (taskset -c 0-7)
  * Using eab and a load testing script
  * The load testing feature of eab involves maintaining a fixed number of long connections to the server and concurrently sending and receiving packets on these connections
* The built-in tnet is enabled by default
* Number of pollers enabled for tnet: 4
* Client-side timeout: Set to 2 seconds by eab
* Server-side timeout: Set to 1 second through framework configuration

## Scenario

Throughput Testing: When the P99 latency of the caller is around 10ms, measure the QPS (Queries Per Second) of the service.


|Mode|	Connections|	QPS/w|	Avery Latency/ms|	P90 Latency/ms|	P99 Latency/ms|	P999 Latency/ms|
|-|-|-|-|-|-|-|
|Synchronous|	100|	486652|	2.67|	4.22	|10.24	|16.76|
|Asynchronous|	100|	404355|	2.61|	4.33	|10.34	|16.07|

## Appendix

```shell
vendor_id	: AuthenticAMD
cpu family	: 23
model		: 49
model name	: AMD EPYC 7K62 48-Core Processor
stepping	: 0
microcode	: 0x1000065
cpu MHz		: 2595.124
cache size	: 512 KB
physical id	: 0
siblings	: 8
core id		: 11
cpu cores	: 8
apicid		: 7
initial apicid	: 7
fpu		: yes
fpu_exception	: yes
cpuid level	: 13
wp		: yes
flags		: fpu vme de pse tsc msr pae mce cx8 apic sep mtrr pge mca cmov pat pse36 clflush mmx fxsr sse sse2 ht syscall nx mmxext fxsr_opt pdpe1gb rdtscp lm art rep_good nopl extd_apicid eagerfpu pni pclmulqdq ssse3 fma cx16 sse4_1 sse4_2 x2apic movbe popcnt aes xsave avx f16c rdrand hypervisor lahf_lm cmp_legacy cr8_legacy abm sse4a misalignsse 3dnowprefetch osvw topoext rsb_ctxsw ibpb vmmcall fsgsbase bmi1 avx2 smep bmi2 rdseed adx smap clflushopt sha_ni xsaveopt xsavec xgetbv1 arat
bogomips	: 5190.24
TLB size	: 1024 4K pages
clflush size	: 64
cache_alignment	: 64
address sizes	: 48 bits physical, 48 bits virtual
power management:
```
