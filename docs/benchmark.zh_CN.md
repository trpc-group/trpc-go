[English](benchmark.md) | 中文

# 性能测试

## 测试环境

* 腾讯云标准型 SA2 CVM，处理器型号 AMD EPYC™ Rome
  * uname -a
    * Linux VM-0-13-centos 3.10.0-1160.92.1.el7.x86_64 #1 SMP Tue Jun 20 11:48:01 UTC 2023 x86_64 x86_64 x86_64 GNU/Linux
  * cat /proc/cpuinfo
    * 见附录（8 核，2.60GHz）
* 内存：16GB
* 网络环境：主调和被调机器间 ping 平均延迟为 0.2ms 左右
* 编译时使用的 Go 版本：1.21.2
* 服务端：绑 8 核（taskset -c 0-7）
  * 逻辑：使用 trpc 协议 echo "hello" 字符串
    * 对应的发包包长 122
    * 对应的收包包长 27
* 客户端：绑 8 核（taskset -c 0-7）
  * 使用 eab 以及压测脚本
  * 压测特点是 eab 会向服务端维持固定数量的长连接，然后在这些长连接上并发收发包
* 默认都是启用的内置 tnet
* 启用 tnet 的 poller 个数： 4
* 客户端超时时间：由 eab 设置 2s
* 服务端超时时间：通过框架配置设置为 1s

## 测试场景

吞吐测试：调用方的 P99 延时在 10ms 左右时，测量服务的 QPS

|模式|	连接数|	QPS/w|	平均时延/ms|	P90时延/ms|	P99时延/ms|	P999时延/ms|
|-|-|-|-|-|-|-|
|同步|	100|	486652|	2.67|	4.22	|10.24	|16.76|
|异步|	100|	404355|	2.61|	4.33	|10.34	|16.07|

## 附录

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

