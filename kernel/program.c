//go:build ignore

//  SPDX-License-Identifier: GPL-2.0
#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/tcp.h>
#include <linux/udp.h>
#include <linux/in.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

#define POLICY_PASS 0
#define POLICY_BLOCK 1
#define POLICY_IGNORE 2

/*
 * Map: sources_count
 * Key:   __u64  — tuple (source IPv4 address (network byte order), dst port) (32, 32 bits)
 * Value: __u64  — packet count
 */
struct
{
    __uint(type, BPF_MAP_TYPE_PERCPU_HASH);
    __uint(max_entries, 4096);
    __type(key, __u64);
    __type(value, __u64);
} sources_count SEC(".maps");

/*
 * Map: tcp_dst_port_count
 * Key:   __u16  — TCP destination port (host byte order)
 * Value: __u64  — packet count per CPU (sum in userspace)
 */
// struct
// {
//     __uint(type, BPF_MAP_TYPE_PERCPU_HASH);
//     __uint(max_entries, 65536);
//     __type(key, __u16);
//     __type(value, __u64);
// } tcp_dst_port_count SEC(".maps");

/*
 * Map: ip_policies
 * Key:   __u32  — source IPv4 address (network byte order)
 * Value: __u64   — policy (block, ignore, etc.)
 */
struct
{
    __uint(type, BPF_MAP_TYPE_PERCPU_HASH);
    __uint(max_entries, 4096);
    __type(key, __u32);
    __type(value, __u64);
} ip_policies SEC(".maps");

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

    /* header bounds check — account for IP header options */
    __u32 ip_hdr_len = ip->ihl * 4;
    if (ip_hdr_len < sizeof(struct iphdr))
        return XDP_PASS;

    // Ensure IP header is within packet bounds
    if ((void *)ip + ip_hdr_len > data_end)
    {
        return XDP_PASS;
    }

    /* Track source IP */
    __u32 src_ip = ip->saddr;

    /* check userland-defined policy */
    __u64 *ip_policy = bpf_map_lookup_elem(&ip_policies, &src_ip);
    if (ip_policy)
    {
        if (*ip_policy == POLICY_BLOCK) // kind of firewall
            return XDP_DROP;
        else if (*ip_policy == POLICY_IGNORE) // ignore, so remove for stat (like bastion)
            return XDP_PASS;
    }

    __u64 key = ((__u64)src_ip << 32); // dst port is 0 for IP-level stats
    __u64 *count = bpf_map_lookup_elem(&sources_count, &key);
    if (count)
    {
        // no atomics needed since the kernel guarantees
        // each CPU only touches its own slot
        (*count)++;
    }
    else
    {
        __u64 init = 1;
        bpf_map_update_elem(&sources_count, &key, &init, BPF_NOEXIST);
    }

    const int transport_bytes = 32;
    /* Only handle TCP and UDP */
    __u64 proto_dst_port = (__u32)ip->protocol << 16;
    if (ip->protocol == IPPROTO_TCP)
    {
        struct tcphdr *tcp = (struct tcphdr *)((unsigned char *)ip + ip_hdr_len);
        if ((void *)tcp + transport_bytes > data_end)
        {
            return XDP_PASS;
        }
        proto_dst_port = proto_dst_port | bpf_ntohs(tcp->dest);
    }
    else if (ip->protocol == IPPROTO_UDP)
    {
        struct udphdr *udp = (struct udphdr *)((unsigned char *)ip + ip_hdr_len);
        if ((void *)udp + transport_bytes > data_end)
        {
            return XDP_PASS;
        }
        proto_dst_port = proto_dst_port | bpf_ntohs(udp->dest);
    }
    else
    {
        /* ignore other protocols */
        return XDP_PASS;
    }

    /* Track TCP destination port (convert to host byte order for the key) */
    // __u16 proto_dst_port = bpf_ntohs(tcp->dest);
    key = key | proto_dst_port; // combine src IP and dst port for finer-grained stats
    count = bpf_map_lookup_elem(&sources_count, &key);
    if (count)
    {
        // no atomics needed since the kernel guarantees
        // each CPU only touches its own slot
        (*count)++;
    }
    else
    {
        __u64 init = 1;
        bpf_map_update_elem(&sources_count, &key, &init, BPF_NOEXIST);
    }

    // __u64 *port_cnt = bpf_map_lookup_elem(&tcp_dst_port_count, &dst_port);
    // if (port_cnt)
    // {
    //     (*port_cnt)++;
    // }
    // else
    // {
    //     __u64 init = 1;
    //     bpf_map_update_elem(&tcp_dst_port_count, &dst_port, &init, BPF_NOEXIST);
    // }

    return XDP_PASS;
}

char _license[] SEC("license") = "GPL";
