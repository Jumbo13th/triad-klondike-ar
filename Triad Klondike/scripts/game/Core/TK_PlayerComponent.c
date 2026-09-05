class TK_PlayerComponentClass : ScriptComponentClass
{
}

// The only client-to-server channel of the mode. Lives on the PlayerController, the
// one entity a client owns, so requests go up as RpcAsk_* and every answer comes
// back to that owner alone as RpcDo_Owner*. Nothing here is replicated by property.
class TK_PlayerComponent : ScriptComponent
{
	// Client-side events the cockpit panel listens to.
	ref ScriptInvoker m_OnCockpitState = new ScriptInvoker();
	ref ScriptInvoker m_OnWallet = new ScriptInvoker();
	ref ScriptInvoker m_OnCommandResult = new ScriptInvoker();
	ref ScriptInvoker m_OnReceipt = new ScriptInvoker();
	ref ScriptInvoker m_OnAuditBegin = new ScriptInvoker();
	ref ScriptInvoker m_OnAuditRow = new ScriptInvoker();
	ref ScriptInvoker m_OnAuditEnd = new ScriptInvoker();

	static TK_PlayerComponent GetByPlayerId(int playerId)
	{
		PlayerManager pm = GetGame().GetPlayerManager();
		if (!pm)
			return null;

		PlayerController pc = pm.GetPlayerController(playerId);
		if (!pc)
			return null;

		return TK_PlayerComponent.Cast(pc.FindComponent(TK_PlayerComponent));
	}

	static TK_PlayerComponent GetLocalInstance()
	{
		PlayerController pc = GetGame().GetPlayerController();
		if (!pc)
			return null;

		return TK_PlayerComponent.Cast(pc.FindComponent(TK_PlayerComponent));
	}

	int GetPlayerId()
	{
		PlayerController pc = PlayerController.Cast(GetOwner());
		if (!pc)
			return -1;

		return pc.GetPlayerId();
	}

	// =====================================================================
	// CLIENT → SERVER
	// =====================================================================

	void AskCockpitState()
	{
		Rpc(RpcAsk_OpenCockpit);
	}

	void AskWallet(string target)
	{
		Rpc(RpcAsk_GetWallet, target);
	}

	void AskCompensate(string opId, string target, int amount, int expectedRevision, string reason)
	{
		Rpc(RpcAsk_Compensate, opId, target, amount, expectedRevision, reason);
	}

	void AskSetConfig(string opId, int answerTimeoutS, int recheckIntervalS, string reason)
	{
		Rpc(RpcAsk_SetBoundaryConfig, opId, answerTimeoutS, recheckIntervalS, reason);
	}

	void AskAudit(int beforeId)
	{
		Rpc(RpcAsk_GetAudit, beforeId);
	}

	[RplRpc(RplChannel.Reliable, RplRcver.Server)]
	protected void RpcAsk_OpenCockpit()
	{
		TK_BackendComponent backend = TK_BackendComponent.GetInstance();
		if (backend)
			backend.SendCockpitState_S(GetPlayerId());
	}

	[RplRpc(RplChannel.Reliable, RplRcver.Server)]
	protected void RpcAsk_GetWallet(string target)
	{
		TK_BackendComponent backend = TK_BackendComponent.GetInstance();
		if (backend)
			backend.RequestWallet_S(GetPlayerId(), target);
	}

	[RplRpc(RplChannel.Reliable, RplRcver.Server)]
	protected void RpcAsk_Compensate(string opId, string target, int amount, int expectedRevision, string reason)
	{
		TK_BackendComponent backend = TK_BackendComponent.GetInstance();
		if (backend)
			backend.Compensate_S(GetPlayerId(), opId, target, amount, expectedRevision, reason);
	}

	[RplRpc(RplChannel.Reliable, RplRcver.Server)]
	protected void RpcAsk_SetBoundaryConfig(string opId, int answerTimeoutS, int recheckIntervalS, string reason)
	{
		TK_BackendComponent backend = TK_BackendComponent.GetInstance();
		if (backend)
			backend.SetConfig_S(GetPlayerId(), opId, answerTimeoutS, recheckIntervalS, reason);
	}

	[RplRpc(RplChannel.Reliable, RplRcver.Server)]
	protected void RpcAsk_GetAudit(int beforeId)
	{
		TK_BackendComponent backend = TK_BackendComponent.GetInstance();
		if (backend)
			backend.RequestAudit_S(GetPlayerId(), beforeId);
	}

	// =====================================================================
	// SERVER → OWNER
	// =====================================================================

	void SendCockpitState_S(int state, string reasonCode, string contractSeen, int configRevision, int answerTimeoutS, int recheckIntervalS, int role)
	{
		Rpc(RpcDo_OwnerCockpitState, state, reasonCode, contractSeen, configRevision, answerTimeoutS, recheckIntervalS, role);
	}

	void SendWallet_S(string target, int total, int reserved, int revision, string reasonCode)
	{
		Rpc(RpcDo_OwnerWallet, target, total, reserved, revision, reasonCode);
	}

	void SendCommandResult_S(string opId, int status, string reasonCode, int total, int reserved, int revision)
	{
		Rpc(RpcDo_OwnerCommandResult, opId, status, reasonCode, total, reserved, revision);
	}

	void SendReceipt_S(string opId, int amount, int totalAfter, string time, string reason)
	{
		Rpc(RpcDo_OwnerReceipt, opId, amount, totalAfter, time, reason);
	}

	void SendAuditBegin_S()
	{
		Rpc(RpcDo_OwnerAuditBegin);
	}

	void SendAuditRow_S(int id, string time, string actorUuid, string type, string target, string outcome, string reasonCode, int amount)
	{
		Rpc(RpcDo_OwnerAuditRow, id, time, actorUuid, type, target, outcome, reasonCode, amount);
	}

	void SendAuditEnd_S(int nextBeforeId, string reasonCode)
	{
		Rpc(RpcDo_OwnerAuditEnd, nextBeforeId, reasonCode);
	}

	[RplRpc(RplChannel.Reliable, RplRcver.Owner)]
	protected void RpcDo_OwnerCockpitState(int state, string reasonCode, string contractSeen, int configRevision, int answerTimeoutS, int recheckIntervalS, int role)
	{
		m_OnCockpitState.Invoke(state, reasonCode, contractSeen, configRevision, answerTimeoutS, recheckIntervalS, role);
	}

	[RplRpc(RplChannel.Reliable, RplRcver.Owner)]
	protected void RpcDo_OwnerWallet(string target, int total, int reserved, int revision, string reasonCode)
	{
		m_OnWallet.Invoke(target, total, reserved, revision, reasonCode);
	}

	[RplRpc(RplChannel.Reliable, RplRcver.Owner)]
	protected void RpcDo_OwnerCommandResult(string opId, int status, string reasonCode, int total, int reserved, int revision)
	{
		m_OnCommandResult.Invoke(opId, status, reasonCode, total, reserved, revision);
	}

	[RplRpc(RplChannel.Reliable, RplRcver.Owner)]
	protected void RpcDo_OwnerReceipt(string opId, int amount, int totalAfter, string time, string reason)
	{
		m_OnReceipt.Invoke(opId, amount, totalAfter, time, reason);
	}

	[RplRpc(RplChannel.Reliable, RplRcver.Owner)]
	protected void RpcDo_OwnerAuditBegin()
	{
		m_OnAuditBegin.Invoke();
	}

	[RplRpc(RplChannel.Reliable, RplRcver.Owner)]
	protected void RpcDo_OwnerAuditRow(int id, string time, string actorUuid, string type, string target, string outcome, string reasonCode, int amount)
	{
		m_OnAuditRow.Invoke(id, time, actorUuid, type, target, outcome, reasonCode, amount);
	}

	[RplRpc(RplChannel.Reliable, RplRcver.Owner)]
	protected void RpcDo_OwnerAuditEnd(int nextBeforeId, string reasonCode)
	{
		m_OnAuditEnd.Invoke(nextBeforeId, reasonCode);
	}
}
