// 作用：定义评分参数。
// 功能：定义对等节点评分的参数和配置，用于评估对等节点的行为和信誉。

package pubsub

import (
	"fmt"
	"math"
	"net"
	"time"

	"github.com/dep2p/go-dep2p/core/peer"
)

// PeerScoreThresholds 包含用于控制对等节点分数的参数
type PeerScoreThresholds struct {
	SkipAtomicValidation        bool    // 是否允许仅设置某些参数而不是所有参数
	GossipThreshold             float64 // 低于该分数时抑制 Gossip 传播，应为负数
	PublishThreshold            float64 // 低于该分数时不应发布消息，应为负数且 <= GossipThreshold
	GraylistThreshold           float64 // 低于该分数时完全抑制消息处理，应为负数且 <= PublishThreshold
	AcceptPXThreshold           float64 // 低于该分数时将忽略 PX，应为正数，限于启动器和其他可信节点
	OpportunisticGraftThreshold float64 // 低于该分数时触发机会性 grafting，应为正数且值小
}

// validate 验证 PeerScoreThresholds 参数
// 返回值:
//   - error: 错误信息
func (p *PeerScoreThresholds) validate() error {
	// 如果没有跳过原子验证，或者 PublishThreshold、GossipThreshold、GraylistThreshold 不为 0
	if !p.SkipAtomicValidation || p.PublishThreshold != 0 || p.GossipThreshold != 0 || p.GraylistThreshold != 0 {
		// 验证 GossipThreshold 是否大于 0 或无效
		if p.GossipThreshold > 0 || isInvalidNumber(p.GossipThreshold) {
			logger.Warnf("GossipThreshold 无效: %f", p.GossipThreshold) // GossipThreshold 无效
			return fmt.Errorf("GossipThreshold 无效: %f", p.GossipThreshold)
		}
		// 验证 PublishThreshold 是否大于 0 或大于 GossipThreshold 或无效
		if p.PublishThreshold > 0 || p.PublishThreshold > p.GossipThreshold || isInvalidNumber(p.PublishThreshold) {
			logger.Warnf("PublishThreshold 无效: %f", p.PublishThreshold) // PublishThreshold 无效
			return fmt.Errorf("PublishThreshold 无效: %f", p.PublishThreshold)
		}
		// 验证 GraylistThreshold 是否大于 0 或大于 PublishThreshold 或无效
		if p.GraylistThreshold > 0 || p.GraylistThreshold > p.PublishThreshold || isInvalidNumber(p.GraylistThreshold) {
			logger.Warnf("GraylistThreshold 无效: %f", p.GraylistThreshold) // GraylistThreshold 无效
			return fmt.Errorf("GraylistThreshold 无效: %f", p.GraylistThreshold)
		}
	}
	// 如果没有跳过原子验证，或者 AcceptPXThreshold 不为 0
	if !p.SkipAtomicValidation || p.AcceptPXThreshold != 0 {
		// 验证 AcceptPXThreshold 是否小于 0 或无效
		if p.AcceptPXThreshold < 0 || isInvalidNumber(p.AcceptPXThreshold) {
			logger.Warnf("AcceptPXThreshold 无效: %f", p.AcceptPXThreshold) // AcceptPXThreshold 无效
			return fmt.Errorf("AcceptPXThreshold 无效: %f", p.AcceptPXThreshold)
		}
	}
	// 如果没有跳过原子验证，或者 OpportunisticGraftThreshold 不为 0
	if !p.SkipAtomicValidation || p.OpportunisticGraftThreshold != 0 {
		// 验证 OpportunisticGraftThreshold 是否小于 0 或无效
		if p.OpportunisticGraftThreshold < 0 || isInvalidNumber(p.OpportunisticGraftThreshold) {
			logger.Warnf("OpportunisticGraftThreshold 无效: %f", p.OpportunisticGraftThreshold) // OpportunisticGraftThreshold 无效
			return fmt.Errorf("OpportunisticGraftThreshold 无效: %f", p.OpportunisticGraftThreshold)
		}
	}
	return nil // 所有验证通过，返回 nil 表示没有错误
}

// PeerScoreParams 包含用于控制对等节点分数的参数
type PeerScoreParams struct {
	SkipAtomicValidation        bool                         // 是否允许仅设置某些参数而不是所有参数
	Topics                      map[string]*TopicScoreParams // 每个主题的分数参数
	TopicScoreCap               float64                      // 主题分数上限
	AppSpecificScore            func(p peer.ID) float64      // 应用程序特定的对等节点分数
	AppSpecificWeight           float64                      // 应用程序特定分数的权重
	IPColocationFactorWeight    float64                      // IP 同位因素的权重
	IPColocationFactorThreshold int                          // IP 同位因素阈值
	IPColocationFactorWhitelist []*net.IPNet                 // IP 同位因素白名单
	BehaviourPenaltyWeight      float64                      // 行为模式处罚的权重
	BehaviourPenaltyThreshold   float64                      // 行为模式处罚的阈值
	BehaviourPenaltyDecay       float64                      // 行为模式处罚的衰减
	DecayInterval               time.Duration                // 参数计数器的衰减间隔
	DecayToZero                 float64                      // 计数器值低于该值时被视为 0
	RetainScore                 time.Duration                // 断开连接的对等节点记住计数器的时间
	SeenMsgTTL                  time.Duration                // 记住消息传递时间
}

// validate 验证 PeerScoreParams 参数
// 返回值:
//   - error: 错误信息
func (p *PeerScoreParams) validate() error {
	// 遍历每个主题及其对应的参数，验证其有效性
	for topic, params := range p.Topics {
		err := params.validate() // 调用主题参数的 validate 方法
		if err != nil {
			logger.Warnf("主题 %s 的评分参数无效: %w", topic, err) // 主题评分参数无效
			return fmt.Errorf("主题 %s 的评分参数无效: %w", topic, err)
		}
	}
	// 如果没有跳过原子验证，或者 TopicScoreCap 不为 0
	if !p.SkipAtomicValidation || p.TopicScoreCap != 0 {
		// 验证 TopicScoreCap 是否小于 0 或无效
		if p.TopicScoreCap < 0 || isInvalidNumber(p.TopicScoreCap) {
			logger.Warnf("TopicScoreCap 无效: %f", p.TopicScoreCap) // TopicScoreCap 无效
			return fmt.Errorf("TopicScoreCap 无效: %f", p.TopicScoreCap)
		}
	}
	// 验证 AppSpecificScore 是否为 nil
	if p.AppSpecificScore == nil {
		if p.SkipAtomicValidation {
			// 如果跳过了原子验证，设置一个默认的应用特定评分函数
			p.AppSpecificScore = func(p peer.ID) float64 {
				return 0
			}
		} else {
			logger.Warnf("缺少应用程序特定的评分函数") // 缺少应用程序特定的评分函数
			return fmt.Errorf("missing application specific score function")
		}
	}
	// 如果没有跳过原子验证，或者 IPColocationFactorWeight 不为 0
	if !p.SkipAtomicValidation || p.IPColocationFactorWeight != 0 {
		// 验证 IPColocationFactorWeight 是否大于 0 或无效
		if p.IPColocationFactorWeight > 0 || isInvalidNumber(p.IPColocationFactorWeight) {
			logger.Warnf("IPColocationFactorWeight 无效: %f", p.IPColocationFactorWeight) // IPColocationFactorWeight 无效
			return fmt.Errorf("IPColocationFactorWeight 无效: %f", p.IPColocationFactorWeight)
		}
		// 验证 IPColocationFactorThreshold 是否小于 1
		if p.IPColocationFactorWeight != 0 && p.IPColocationFactorThreshold < 1 {
			logger.Warnf("IPColocationFactorThreshold 无效: %d", p.IPColocationFactorThreshold) // IPColocationFactorThreshold 无效
			return fmt.Errorf("IPColocationFactorThreshold 无效: %d", p.IPColocationFactorThreshold)
		}
	}
	// 如果没有跳过原子验证，或者 BehaviourPenaltyWeight 或 BehaviourPenaltyThreshold 不为 0
	if !p.SkipAtomicValidation || p.BehaviourPenaltyWeight != 0 || p.BehaviourPenaltyThreshold != 0 {
		// 验证 BehaviourPenaltyWeight 是否大于 0 或无效
		if p.BehaviourPenaltyWeight > 0 || isInvalidNumber(p.BehaviourPenaltyWeight) {
			logger.Warnf("BehaviourPenaltyWeight 无效: %f", p.BehaviourPenaltyWeight) // BehaviourPenaltyWeight 无效
			return fmt.Errorf("BehaviourPenaltyWeight 无效: %f", p.BehaviourPenaltyWeight)
		}
		// 验证 BehaviourPenaltyDecay 是否不在 (0, 1) 区间内或无效
		if p.BehaviourPenaltyWeight != 0 && (p.BehaviourPenaltyDecay <= 0 || p.BehaviourPenaltyDecay >= 1 || isInvalidNumber(p.BehaviourPenaltyDecay)) {
			logger.Warnf("BehaviourPenaltyDecay 无效: %f", p.BehaviourPenaltyDecay) // BehaviourPenaltyDecay 无效
			return fmt.Errorf("BehaviourPenaltyDecay 无效: %f", p.BehaviourPenaltyDecay)
		}
		// 验证 BehaviourPenaltyThreshold 是否小于 0 或无效
		if p.BehaviourPenaltyThreshold < 0 || isInvalidNumber(p.BehaviourPenaltyThreshold) {
			logger.Warnf("BehaviourPenaltyThreshold 无效: %f", p.BehaviourPenaltyThreshold) // BehaviourPenaltyThreshold 无效
			return fmt.Errorf("BehaviourPenaltyThreshold 无效: %f", p.BehaviourPenaltyThreshold)
		}
	}
	// 如果没有跳过原子验证，或者 DecayInterval 或 DecayToZero 不为 0
	if !p.SkipAtomicValidation || p.DecayInterval != 0 || p.DecayToZero != 0 {
		// 验证 DecayInterval 是否小于 1 秒
		if p.DecayInterval < time.Second {
			logger.Warnf("DecayInterval 无效: %s", p.DecayInterval) // DecayInterval 无效
			return fmt.Errorf("DecayInterval 无效: %s", p.DecayInterval)
		}
		// 验证 DecayToZero 是否不在 (0, 1) 区间内或无效
		if p.DecayToZero <= 0 || p.DecayToZero >= 1 || isInvalidNumber(p.DecayToZero) {
			logger.Warnf("DecayToZero 无效: %f", p.DecayToZero) // DecayToZero 无效
			return fmt.Errorf("DecayToZero 无效: %f", p.DecayToZero)
		}
	}
	return nil // 所有验证通过，返回 nil 表示没有错误
}

// TopicScoreParams 包含用于控制主题分数的参数
type TopicScoreParams struct {
	SkipAtomicValidation            bool          // 是否允许仅设置某些参数而不是所有参数
	TopicWeight                     float64       // 主题权重
	TimeInMeshWeight                float64       // 在 mesh 中的时间权重
	TimeInMeshQuantum               time.Duration // 在 mesh 中的时间量子
	TimeInMeshCap                   float64       // 在 mesh 中的时间上限
	FirstMessageDeliveriesWeight    float64       // 首次消息传递的权重
	FirstMessageDeliveriesDecay     float64       // 首次消息传递的衰减
	FirstMessageDeliveriesCap       float64       // 首次消息传递的上限
	MeshMessageDeliveriesWeight     float64       // mesh 消息传递的权重
	MeshMessageDeliveriesDecay      float64       // mesh 消息传递的衰减
	MeshMessageDeliveriesCap        float64       // mesh 消息传递的上限
	MeshMessageDeliveriesThreshold  float64       // mesh 消息传递的阈值
	MeshMessageDeliveriesWindow     time.Duration // mesh 消息传递的窗口
	MeshMessageDeliveriesActivation time.Duration // mesh 消息传递的激活时间
	MeshFailurePenaltyWeight        float64       // mesh 失败处罚的权重
	MeshFailurePenaltyDecay         float64       // mesh 失败处罚的衰减
	InvalidMessageDeliveriesWeight  float64       // 无效消息传递的权重
	InvalidMessageDeliveriesDecay   float64       // 无效消息传递的衰减
}

// validate 验证 TopicScoreParams 参数
// 返回值:
//   - error: 错误信息
func (p *TopicScoreParams) validate() error {
	if p.TopicWeight < 0 || isInvalidNumber(p.TopicWeight) {
		logger.Warnf("TopicWeight 无效: %f", p.TopicWeight) // TopicWeight 无效
		return fmt.Errorf("TopicWeight 无效: %f", p.TopicWeight)
	}
	if err := p.validateTimeInMeshParams(); err != nil {
		return err
	}
	if err := p.validateMessageDeliveryParams(); err != nil {
		return err
	}
	if err := p.validateMeshMessageDeliveryParams(); err != nil {
		return err
	}
	if err := p.validateMessageFailurePenaltyParams(); err != nil {
		return err
	}
	if err := p.validateInvalidMessageDeliveryParams(); err != nil {
		return err
	}
	return nil
}

// validateTimeInMeshParams 验证 TimeInMesh 参数
// 返回值:
//   - error: 错误信息
func (p *TopicScoreParams) validateTimeInMeshParams() error {
	if p.SkipAtomicValidation {
		if p.TimeInMeshWeight == 0 && p.TimeInMeshQuantum == 0 && p.TimeInMeshCap == 0 {
			return nil
		}
	}
	if p.TimeInMeshQuantum == 0 {
		logger.Warnf("TimeInMeshQuantum 无效: %s", p.TimeInMeshQuantum) // TimeInMeshQuantum 无效
		return fmt.Errorf("TimeInMeshQuantum 无效: %s", p.TimeInMeshQuantum)
	}
	if p.TimeInMeshWeight < 0 || isInvalidNumber(p.TimeInMeshWeight) {
		logger.Warnf("TimeInMeshWeight 无效: %f", p.TimeInMeshWeight) // TimeInMeshWeight 无效
		return fmt.Errorf("TimeInMeshWeight 无效: %f", p.TimeInMeshWeight)
	}
	if p.TimeInMeshWeight != 0 && p.TimeInMeshQuantum <= 0 {
		logger.Warnf("TimeInMeshQuantum 无效: %s", p.TimeInMeshQuantum) // TimeInMeshQuantum 无效
		return fmt.Errorf("TimeInMeshQuantum 无效: %s", p.TimeInMeshQuantum)
	}
	if p.TimeInMeshWeight != 0 && (p.TimeInMeshCap <= 0 || isInvalidNumber(p.TimeInMeshCap)) {
		logger.Warnf("TimeInMeshCap 无效: %f", p.TimeInMeshCap) // TimeInMeshCap 无效
		return fmt.Errorf("TimeInMeshCap 无效: %f", p.TimeInMeshCap)
	}
	return nil
}

// validateMessageDeliveryParams 验证 FirstMessageDeliveries 参数
// 返回值:
//   - error: 错误信息
func (p *TopicScoreParams) validateMessageDeliveryParams() error {
	if p.SkipAtomicValidation {
		if p.FirstMessageDeliveriesWeight == 0 && p.FirstMessageDeliveriesCap == 0 && p.FirstMessageDeliveriesDecay == 0 {
			return nil
		}
	}
	if p.FirstMessageDeliveriesWeight < 0 || isInvalidNumber(p.FirstMessageDeliveriesWeight) {
		logger.Warnf("FirstMessageDeliveriesWeight 无效: %f", p.FirstMessageDeliveriesWeight) // FirstMessageDeliveriesWeight 无效
		return fmt.Errorf("FirstMessageDeliveriesWeight 无效: %f", p.FirstMessageDeliveriesWeight)
	}
	if p.FirstMessageDeliveriesWeight != 0 && (p.FirstMessageDeliveriesDecay <= 0 || p.FirstMessageDeliveriesDecay >= 1 || isInvalidNumber(p.FirstMessageDeliveriesDecay)) {
		logger.Warnf("FirstMessageDeliveriesDecay 无效: %f", p.FirstMessageDeliveriesDecay) // FirstMessageDeliveriesDecay 无效
		return fmt.Errorf("FirstMessageDeliveriesDecay 无效: %f", p.FirstMessageDeliveriesDecay)
	}
	if p.FirstMessageDeliveriesWeight != 0 && (p.FirstMessageDeliveriesCap <= 0 || isInvalidNumber(p.FirstMessageDeliveriesCap)) {
		logger.Warnf("FirstMessageDeliveriesCap 无效: %f", p.FirstMessageDeliveriesCap) // FirstMessageDeliveriesCap 无效
		return fmt.Errorf("FirstMessageDeliveriesCap 无效: %f", p.FirstMessageDeliveriesCap)
	}
	return nil
}

// validateMeshMessageDeliveryParams 验证 MeshMessageDeliveries 参数
// 返回值:
//   - error: 错误信息
func (p *TopicScoreParams) validateMeshMessageDeliveryParams() error {
	if p.SkipAtomicValidation {
		if p.MeshMessageDeliveriesWeight == 0 &&
			p.MeshMessageDeliveriesCap == 0 &&
			p.MeshMessageDeliveriesDecay == 0 &&
			p.MeshMessageDeliveriesThreshold == 0 &&
			p.MeshMessageDeliveriesWindow == 0 &&
			p.MeshMessageDeliveriesActivation == 0 {
			return nil
		}
	}
	if p.MeshMessageDeliveriesWeight > 0 || isInvalidNumber(p.MeshMessageDeliveriesWeight) {
		logger.Warnf("MeshMessageDeliveriesWeight 无效: %f", p.MeshMessageDeliveriesWeight) // MeshMessageDeliveriesWeight 无效
		return fmt.Errorf("MeshMessageDeliveriesWeight 无效: %f", p.MeshMessageDeliveriesWeight)
	}
	if p.MeshMessageDeliveriesWeight != 0 && (p.MeshMessageDeliveriesDecay <= 0 || p.MeshMessageDeliveriesDecay >= 1 || isInvalidNumber(p.MeshMessageDeliveriesDecay)) {
		logger.Warnf("MeshMessageDeliveriesDecay 无效: %f", p.MeshMessageDeliveriesDecay) // MeshMessageDeliveriesDecay 无效
		return fmt.Errorf("MeshMessageDeliveriesDecay 无效: %f", p.MeshMessageDeliveriesDecay)
	}
	if p.MeshMessageDeliveriesWeight != 0 && (p.MeshMessageDeliveriesCap <= 0 || isInvalidNumber(p.MeshMessageDeliveriesCap)) {
		logger.Warnf("MeshMessageDeliveriesCap 无效: %f", p.MeshMessageDeliveriesCap) // MeshMessageDeliveriesCap 无效
		return fmt.Errorf("MeshMessageDeliveriesCap 无效: %f", p.MeshMessageDeliveriesCap)
	}
	if p.MeshMessageDeliveriesWeight != 0 && (p.MeshMessageDeliveriesThreshold <= 0 || isInvalidNumber(p.MeshMessageDeliveriesThreshold)) {
		logger.Warnf("MeshMessageDeliveriesThreshold 无效: %f", p.MeshMessageDeliveriesThreshold) // MeshMessageDeliveriesThreshold 无效
		return fmt.Errorf("MeshMessageDeliveriesThreshold 无效: %f", p.MeshMessageDeliveriesThreshold)
	}
	if p.MeshMessageDeliveriesWindow < 0 {
		logger.Warnf("MeshMessageDeliveriesWindow 无效: %s", p.MeshMessageDeliveriesWindow) // MeshMessageDeliveriesWindow 无效
		return fmt.Errorf("MeshMessageDeliveriesWindow 无效: %s", p.MeshMessageDeliveriesWindow)
	}
	if p.MeshMessageDeliveriesWeight != 0 && p.MeshMessageDeliveriesActivation < time.Second {
		logger.Warnf("MeshMessageDeliveriesActivation 无效: %s", p.MeshMessageDeliveriesActivation) // MeshMessageDeliveriesActivation 无效
		return fmt.Errorf("MeshMessageDeliveriesActivation 无效: %s", p.MeshMessageDeliveriesActivation)
	}
	return nil
}

// validateMessageFailurePenaltyParams 验证 MeshFailurePenalty 参数
// 返回值:
//   - error: 错误信息
func (p *TopicScoreParams) validateMessageFailurePenaltyParams() error {
	if p.SkipAtomicValidation {
		if p.MeshFailurePenaltyDecay == 0 && p.MeshFailurePenaltyWeight == 0 {
			return nil
		}
	}
	if p.MeshFailurePenaltyWeight > 0 || isInvalidNumber(p.MeshFailurePenaltyWeight) {
		logger.Warnf("MeshFailurePenaltyWeight 无效: %f", p.MeshFailurePenaltyWeight) // MeshFailurePenaltyWeight 无效
		return fmt.Errorf("MeshFailurePenaltyWeight 无效: %f", p.MeshFailurePenaltyWeight)
	}
	if p.MeshFailurePenaltyWeight != 0 && (isInvalidNumber(p.MeshFailurePenaltyDecay) || p.MeshFailurePenaltyDecay <= 0 || p.MeshFailurePenaltyDecay >= 1) {
		logger.Warnf("MeshFailurePenaltyDecay 无效: %f", p.MeshFailurePenaltyDecay) // MeshFailurePenaltyDecay 无效
		return fmt.Errorf("MeshFailurePenaltyDecay 无效: %f", p.MeshFailurePenaltyDecay)
	}
	return nil
}

// validateInvalidMessageDeliveryParams 验证 InvalidMessageDeliveries 参数
// 返回值:
//   - error: 错误信息
func (p *TopicScoreParams) validateInvalidMessageDeliveryParams() error {
	if p.SkipAtomicValidation {
		if p.InvalidMessageDeliveriesDecay == 0 && p.InvalidMessageDeliveriesWeight == 0 {
			return nil
		}
	}
	if p.InvalidMessageDeliveriesWeight > 0 || isInvalidNumber(p.InvalidMessageDeliveriesWeight) {
		logger.Warnf("InvalidMessageDeliveriesWeight 无效: %f", p.InvalidMessageDeliveriesWeight) // InvalidMessageDeliveriesWeight 无效
		return fmt.Errorf("InvalidMessageDeliveriesWeight 无效: %f", p.InvalidMessageDeliveriesWeight)
	}
	if p.InvalidMessageDeliveriesDecay <= 0 || p.InvalidMessageDeliveriesDecay >= 1 || isInvalidNumber(p.InvalidMessageDeliveriesDecay) {
		logger.Warnf("InvalidMessageDeliveriesDecay 无效: %f", p.InvalidMessageDeliveriesDecay) // InvalidMessageDeliveriesDecay 无效
		return fmt.Errorf("InvalidMessageDeliveriesDecay 无效: %f", p.InvalidMessageDeliveriesDecay)
	}
	return nil
}

const (
	DefaultDecayInterval = time.Second // 默认的衰减间隔
	DefaultDecayToZero   = 0.01        // 默认的衰减到零值
)

// ScoreParameterDecay 计算参数的衰减因子，假设 DecayInterval 为 1s 并且值在低于 0.01 时衰减到零
// 参数:
//   - decay: 衰减时间
//
// 返回值:
//   - float64: 衰减因子
func ScoreParameterDecay(decay time.Duration) float64 {
	return ScoreParameterDecayWithBase(decay, DefaultDecayInterval, DefaultDecayToZero)
}

// ScoreParameterDecayWithBase 使用基准 DecayInterval 计算参数的衰减因子
// 参数:
//   - decay: 衰减时间
//   - base: 基准衰减间隔
//   - decayToZero: 衰减到零值
//
// 返回值:
//   - float64: 衰减因子
func ScoreParameterDecayWithBase(decay time.Duration, base time.Duration, decayToZero float64) float64 {
	ticks := float64(decay / base)
	return math.Pow(decayToZero, 1/ticks)
}

// isInvalidNumber 检查提供的浮点数是否为 NaN 或无穷大
// 参数:
//   - num: 要检查的浮点数
//
// 返回值:
//   - bool: 是否为无效数字
func isInvalidNumber(num float64) bool {
	return math.IsNaN(num) || math.IsInf(num, 0)
}
