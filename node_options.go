package pubsub

import (
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// PubSubType 定义发布订阅的类型
type PubSubType int

const (
	GossipSub PubSubType = iota // GossipSub 类型
	FloodSub                    // FloodSub 类型
	RandomSub                   // RandomSub 类型
)

// Options 定义了 PubSub 的配置选项
type Options struct {
	mu sync.Mutex // 互斥锁，用于保护字段的并发访问

	FollowupTime        time.Duration   // 跟随时间
	GossipFactor        float64         // Gossip 因子
	D                   int             // GossipSub 主题网格的理想度数
	Dlo                 int             // GossipSub 主题网格中保持的最少节点数
	MaxPendingConns     int             // 最大待处理连接数
	MaxMessageSize      int             // 最大消息大小
	SignMessages        bool            // 是否签名消息
	ValidateMessages    bool            // 是否验证消息
	DirectPeers         []peer.AddrInfo // 直连对等节点列表
	HeartbeatInterval   time.Duration   // 心跳间隔
	MaxTransmissionSize int             // 最大传输大小
	LoadConfig          bool            // 是否加载配置选项
	PubSubMode          PubSubType      // 发布订阅模式
}

// NodeOption 定义了一个函数类型，用于配置PubSub
type NodeOption func(*Options) error

// ApplyOptions 应用给定的选项到 Options 对象
// 参数:
//   - opts: 可变参数，包含多个 NodeOption 函数
//
// 返回值:
//   - error: 如果应用选项时出现错误，返回相应的错误信息
func (opt *Options) ApplyOptions(opts ...NodeOption) error {
	opt.mu.Lock()         // 加锁以保护并发访问
	defer opt.mu.Unlock() // 函数结束时解锁
	for _, o := range opts {
		if err := o(opt); err != nil {
			return err // 如果应用某个选项出错，立即返回错误
		}
	}
	return nil // 所有选项应用成功，返回 nil
}

// DefaultOptions 返回一个带有默认配置的 Options 对象
// 返回值:
//   - *Options: 包含默认配置的 Options 对象
// func DefaultOptions() *Options {
// 	return &Options{
// 		FollowupTime:        1 * time.Second, // 默认跟随时间为1秒
// 		GossipFactor:        0.12,            // 默认Gossip因子为0.12
// 		D:                   8,               // 默认主题网格的理想度数为8
// 		Dlo:                 6,               // 默认主题网格中保持的最少节点数为6
// 		MaxPendingConns:     23,              // 默认最大待处理连接数为23
// 		MaxMessageSize:      1024 * 1024,     // 默认最大消息大小为1MB
// 		SignMessages:        true,            // 默认签名消息
// 		ValidateMessages:    true,            // 默认验证消息
// 		HeartbeatInterval:   1 * time.Second, // 默认心跳间隔为1秒
// 		MaxTransmissionSize: 10 << 20,        // 默认最大传输大小为10MB
// 		LoadConfig:          false,           // 默认不加载配置
// 		PubSubMode:          GossipSub,       // 默认使用 GossipSub
// 	}
// }

// DefaultOptions 返回一个带有默认配置的 Options 对象
// 返回值:
//   - *Options: 包含默认配置的 Options 对象
func DefaultOptions() *Options {
	return &Options{
		// 降低跟随时间，加快消息响应速度
		// 在小规模网络中，消息传播延迟较小，可以使用更短的跟随时间
		FollowupTime: 500 * time.Millisecond,

		// 降低 Gossip 因子，减少不必要的消息传播
		// 在节点数量少的情况下，不需要太高的传播率
		GossipFactor: 0.25, // 提高到 0.25，因为只有 2 个节点

		// 降低网格度数要求，适应小规模网络
		// D 是每个主题维护的最大对等点数量
		// 对于 2 个节点的网络，设置为 2 就足够了
		D: 2, // 最小化，因为只有 2 个节点

		// Dlo 是维护的最小对等点数量
		// 对于 2 个节点的网络，设置为 1 确保至少保持一个连接
		Dlo: 1, // 最小化，确保至少有一个连接

		// 减少最大待处理连接数，因为节点数量少
		MaxPendingConns: 5, // 小规模网络不需要太多待处理连接

		// 保持合理的消息大小限制
		MaxMessageSize: 1024 * 1024, // 1MB，根据实际需求调整

		// 在小规模可信网络中，可以考虑关闭签名和验证以提高性能
		SignMessages:     false, // 小规模可信网络可以关闭
		ValidateMessages: false, // 小规模可信网络可以关闭

		// 降低心跳间隔，加快节点状态更新
		// 在小规模网络中，可以使用更频繁的心跳来保持连接状态
		HeartbeatInterval: 500 * time.Millisecond,

		// 保持合理的传输大小限制
		MaxTransmissionSize: 5 << 20, // 5MB，根据实际需求调整

		// 启用配置加载，使用自定义的优化配置
		LoadConfig: true, // 启用配置以使用优化的参数

		// 使用 GossipSub 模式
		// 即使是小规模网络，GossipSub 也能提供可靠的消息传递
		PubSubMode: GossipSub,
	}
}

// WithSetFollowupTime 设置跟随时间
// 参数:
//   - t: 要设置的跟随时间
//
// 返回值:
//   - NodeOption: 返回一个配置函数
func WithSetFollowupTime(t time.Duration) NodeOption {
	return func(o *Options) error {
		o.FollowupTime = t
		return nil
	}
}

// WithSetGossipFactor 设置Gossip因子
// 参数:
//   - f: 要设置的Gossip因子
//
// 返回值:
//   - NodeOption: 返回一个配置函数
func WithSetGossipFactor(f float64) NodeOption {
	return func(o *Options) error {
		o.GossipFactor = f
		return nil
	}
}

// WithSetMaxPendingConns 设置最大待处理连接数
// 参数:
//   - n: 要设置的最大待处理连接数
//
// 返回值:
//   - NodeOption: 返回一个配置函数
func WithSetMaxPendingConns(n int) NodeOption {
	return func(o *Options) error {
		o.MaxPendingConns = n
		return nil
	}
}

// WithSetMaxMessageSize 设置最大消息大小
// 参数:
//   - size: 要设置的最大消息大小
//
// 返回值:
//   - NodeOption: 返回一个配置函数
func WithSetMaxMessageSize(size int) NodeOption {
	return func(o *Options) error {
		o.MaxMessageSize = size
		return nil
	}
}

// WithSetSignMessages 设置是否签名消息
// 参数:
//   - sign: 是否签名消息
//
// 返回值:
//   - NodeOption: 返回一个配置函数
func WithSetSignMessages(sign bool) NodeOption {
	return func(o *Options) error {
		o.SignMessages = sign
		return nil
	}
}

// WithSetValidateMessages 设置是否验证消息
// 参数:
//   - validate: 是否验证消息
//
// 返回值:
//   - NodeOption: 返回一个配置函数
func WithSetValidateMessages(validate bool) NodeOption {
	return func(o *Options) error {
		o.ValidateMessages = validate
		return nil
	}
}

// WithSetDirectPeers 设置直连对等节点列表
// 参数:
//   - peers: 要设置的直连对等节点列表
//
// 返回值:
//   - NodeOption: 返回一个配置函数
func WithSetDirectPeers(peers []peer.AddrInfo) NodeOption {
	return func(o *Options) error {
		o.DirectPeers = peers
		return nil
	}
}

// WithSetHeartbeatInterval 设置心跳间隔
// 参数:
//   - interval: 要设置的心跳间隔
//
// 返回值:
//   - NodeOption: 返回一个配置函数
func WithSetHeartbeatInterval(interval time.Duration) NodeOption {
	return func(o *Options) error {
		o.HeartbeatInterval = interval
		return nil
	}
}

// WithSetMaxTransmissionSize 设置最大传输大小
// 参数:
//   - size: 要设置的最大传输大小
//
// 返回值:
//   - NodeOption: 返回一个配置函数
func WithSetMaxTransmissionSize(size int) NodeOption {
	return func(o *Options) error {
		o.MaxTransmissionSize = size
		return nil
	}
}

// WithSetD 设置 GossipSub 主题网格的理想度数
// 参数:
//   - d: 要设置的理想度数
//
// 返回值:
//   - NodeOption: 返回一个配置函数
func WithSetD(d int) NodeOption {
	return func(o *Options) error {
		o.D = d
		return nil
	}
}

// WithSetDlo 设置 GossipSub 主题网格中保持的最少节点数
// 参数:
//   - dlo: 要设置的最少节点数
//
// 返回值:
//   - NodeOption: 返回一个配置函数
func WithSetDlo(dlo int) NodeOption {
	return func(o *Options) error {
		o.Dlo = dlo
		return nil
	}
}

// WithSetLoadConfig 设置是否加载配置选项
func WithSetLoadConfig(load bool) NodeOption {
	return func(o *Options) error {
		o.LoadConfig = load
		return nil
	}
}

// WithSetPubSubMode 设置发布订阅模式
func WithSetPubSubMode(mode PubSubType) NodeOption {
	return func(o *Options) error {
		o.PubSubMode = mode
		return nil
	}
}

// 以下是获取各种选项值的方法，它们都使用互斥锁来保证并发安全

// GetFollowupTime 获取跟随时间
// 返回值:
//   - time.Duration: 当前设置的跟随时间
func (o *Options) GetFollowupTime() time.Duration {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.FollowupTime
}

// GetGossipFactor 获取Gossip因子
// 返回值:
//   - float64: 当前设置的Gossip因子
func (o *Options) GetGossipFactor() float64 {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.GossipFactor
}

// GetMaxPendingConns 获取最大待处理连接数
// 返回值:
//   - int: 当前设置的最大待处理连接数
func (o *Options) GetMaxPendingConns() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.MaxPendingConns
}

// GetMaxMessageSize 获取最大消息大小
// 返回值:
//   - int: 当前设置的最大消息大小
func (o *Options) GetMaxMessageSize() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.MaxMessageSize
}

// GetSignMessages 获取是否签名消息
// 返回值:
//   - bool: 当前是否设置为签名消息
func (o *Options) GetSignMessages() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.SignMessages
}

// GetValidateMessages 获取是否验证消息
// 返回值:
//   - bool: 当前是否设置为验证消息
func (o *Options) GetValidateMessages() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.ValidateMessages
}

// GetDirectPeers 获取直连对等节点列表
// 返回值:
//   - []peer.AddrInfo: 当前设置的直连对等节点列表
func (o *Options) GetDirectPeers() []peer.AddrInfo {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.DirectPeers
}

// GetHeartbeatInterval 获取心跳间隔
// 返回值:
//   - time.Duration: 当前设置的心跳间隔
func (o *Options) GetHeartbeatInterval() time.Duration {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.HeartbeatInterval
}

// GetMaxTransmissionSize 获取最大传输大小
// 返回值:
//   - int: 当前设置的最大传输大小
func (o *Options) GetMaxTransmissionSize() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.MaxTransmissionSize
}

// GetD 获取 GossipSub 主题网格的理想度数
// 返回值:
//   - int: 当前设置的理想度数
func (o *Options) GetD() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.D
}

// GetDlo 获取 GossipSub 主题网格中保持的最少节点数
// 返回值:
//   - int: 当前设置的最少节点数
func (o *Options) GetDlo() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.Dlo
}

// GetLoadConfig 获取是否加载配置选项
func (o *Options) GetLoadConfig() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.LoadConfig
}

// GetPubSubMode 获取发布订阅模式
func (o *Options) GetPubSubMode() PubSubType {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.PubSubMode
}