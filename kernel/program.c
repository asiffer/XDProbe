//go:build ignore

//  SPDX-License-Identifier: GPL-2.0
#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/tcp.h>
#include <linux/in.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

/*
 * Map: src_ip_count
 * Key:   __u32  — source IPv4 address (network byte order)
 * Value: __u64  — packet count
 */
struct
{
    __uint(type, BPF_MAP_TYPE_PERCPU_HASH);
    __uint(max_entries, 4096);
    __type(key, __u32);
    __type(value, __u64);
} src_ip_count SEC(".maps");

/*
 * Map: tcp_dst_port_count
 * Key:   __u16  — TCP destination port (host byte order)
 * Value: __u64  — packet count per CPU (sum in userspace)
 */
struct
{
    __uint(type, BPF_MAP_TYPE_PERCPU_HASH);
    __uint(max_entries, 65536);
    __type(key, __u16);
    __type(value, __u64);
} tcp_dst_port_count SEC(".maps");

SEC("xdp")
int xdprobe(struct xdp_md *ctx)
{
    void *data = (void *)(long)ctx->data;
    void *data_end = (void *)(long)ctx->data_end;

    /* Ethernet header bounds check */
    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end)
        return XDP_PASS;

    /* Only handle IPv4 */
    if (bpf_ntohs(eth->h_proto) != ETH_P_IP)
        return XDP_PASS;

    /* IP header bounds check */
    struct iphdr *ip = (void *)(eth + 1);
    if ((void *)(ip + 1) > data_end)
        return XDP_PASS;

    /* Track source IP */
    __u32 src_ip = ip->saddr;
    __u64 *ip_cnt = bpf_map_lookup_elem(&src_ip_count, &src_ip);
    if (ip_cnt)
    {
        // no atomics needed since the kernel guarantees
        // each CPU only touches its own slot
        (*ip_cnt)++;
    }
    else
    {
        __u64 init = 1;
        bpf_map_update_elem(&src_ip_count, &src_ip, &init, BPF_NOEXIST);
    }

    /* Only handle TCP */
    if (ip->protocol != IPPROTO_TCP)
        return XDP_PASS;

    /* TCP header bounds check — account for IP header options */
    __u32 ip_hdr_len = ip->ihl * 4;
    if (ip_hdr_len < sizeof(*ip))
        return XDP_PASS;

    struct tcphdr *tcp = (void *)ip + ip_hdr_len;
    if ((void *)(tcp + 1) > data_end)
        return XDP_PASS;

    /* Track TCP destination port (convert to host byte order for the key) */
    __u16 dst_port = bpf_ntohs(tcp->dest);
    __u64 *port_cnt = bpf_map_lookup_elem(&tcp_dst_port_count, &dst_port);
    if (port_cnt)
    {
        (*port_cnt)++;
    }
    else
    {
        __u64 init = 1;
        bpf_map_update_elem(&tcp_dst_port_count, &dst_port, &init, BPF_NOEXIST);
    }

    return XDP_PASS;
}

char _license[] SEC("license") = "GPL";
