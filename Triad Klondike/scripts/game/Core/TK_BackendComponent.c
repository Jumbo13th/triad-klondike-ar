// $profile:TK_Backend.json — the only backend setting the game server holds. The
// rest (operators, runtime values) is the backend's and arrives with each health poll.
class TK_BackendConfig : JsonApiStruct
{
	string BaseUrl = "http://127.0.0.1:8471/";

	void TK_BackendConfig()
	{
		RegV("BaseUrl");
	}
}

enum TK_EBoundaryState
{
	NOT_CHECKED,
	READY,
	UNREACHABLE,
	VERSION_UNKNOWN,
}

enum TK_ERequestKind
{
	HEALTH,
	CONNECT,
	WALLET,
	COMPENSATE,
	CONFIG,
	SECURITY,
	AUDIT,
}

// One in-flight HTTP request. The engine deletes a RestCallback that nothing holds
// a strong reference to before the answer arrives, so the component keeps these
// until their callback fires.
class TK_BackendRequest : Managed
{
	TK_ERequestKind m_eKind;
	int m_iPlayerId;
	string m_sOpId;
	string m_sTargetUuid;
	int m_iExpectedRevision;
	ref RestCallback m_Callback;
	TK_BackendComponent m_Component;

	void Send(notnull TK_BackendComponent component, notnull RestContext context, string path, string body)
	{
		m_Component = component;
		m_Callback = new RestCallback();
		m_Callback.SetOnSuccess(OnSuccess);
		m_Callback.SetOnError(OnError);
		if (body == "")
			context.GET(m_Callback, path);
		else
			context.POST(m_Callback, path, body);
	}

	protected void OnSuccess(RestCallback cb)
	{
		if (m_Component)
			m_Component.OnRequestDone(this, cb);
	}

	protected void OnError(RestCallback cb)
	{
		if (m_Component)
			m_Component.OnRequestDone(this, cb);
	}
}

// What the server knows about a connected player once the backend confirmed them.
class TK_PlayerSession : Managed
{
	string m_sUuid;
	string m_sRole;
	string m_sPendingOpId;
}

class TK_BackendComponentClass : SCR_BaseGameModeComponentClass
{
}

// The boundary to the local backend: contract check, readiness, and every call the
// game makes. Server only; nothing here is replicated. Owner RPCs to players go
// through TK_PlayerComponent.
class TK_BackendComponent : SCR_BaseGameModeComponent
{
	protected static const string CONFIG_PATH = "$profile:TK_Backend.json";
	protected static const string ROLE_OPERATOR = "operator";
	protected static const int AUDIT_PAGE = 20;

	protected static TK_BackendComponent s_Instance;

	protected RestContext m_Context;
	protected ref array<ref TK_BackendRequest> m_aPending = {};
	protected ref map<int, ref TK_PlayerSession> m_mSessions = new map<int, ref TK_PlayerSession>();

	protected TK_EBoundaryState m_eState = TK_EBoundaryState.NOT_CHECKED;
	protected string m_sContractSeen;
	protected int m_iConfigRevision;
	protected int m_iAnswerTimeoutS = 5;
	protected int m_iRecheckIntervalS = 15;

	static TK_BackendComponent GetInstance()
	{
		return s_Instance;
	}

	void ~TK_BackendComponent()
	{
		if (s_Instance == this)
			s_Instance = null;
	}

	override void OnPostInit(IEntity owner)
	{
		super.OnPostInit(owner);

		if (!Replication.IsServer())
			return;

		s_Instance = this;

		TK_BackendConfig config = new TK_BackendConfig();
		if (FileIO.FileExists(CONFIG_PATH))
			config.LoadFromFile(CONFIG_PATH);
		else
			config.SaveToFile(CONFIG_PATH);

		string baseUrl = config.BaseUrl;
		if (!baseUrl.EndsWith("/"))
			baseUrl += "/";

		m_Context = GetGame().GetRestApi().GetContext(baseUrl);
		if (!m_Context)
		{
			Print(string.Format("[TK] backend: no REST context for '%1'; the boundary stays unreachable", baseUrl), LogLevel.ERROR);
			return;
		}
		m_Context.SetTimeout(m_iAnswerTimeoutS);

		GetGame().GetCallqueue().CallLater(PollHealth_S, 1000, false);
	}

	// =====================================================================
	// READINESS
	// =====================================================================

	TK_EBoundaryState GetState()
	{
		return m_eState;
	}

	bool IsReady_S()
	{
		return m_eState == TK_EBoundaryState.READY;
	}

	protected string NotReadyReason_S()
	{
		if (m_eState == TK_EBoundaryState.VERSION_UNKNOWN)
			return TK_Reason.CONTRACT_VERSION_UNKNOWN;
		return TK_Reason.BACKEND_UNREACHABLE;
	}

	// Polled always, not only while down, so a backend that dies between player
	// commands is noticed within one interval. Re-armed after each poll so a changed
	// interval applies at once.
	protected void PollHealth_S()
	{
		if (!m_Context)
			return;

		TK_BackendRequest request = new TK_BackendRequest();
		request.m_eKind = TK_ERequestKind.HEALTH;
		request.m_iPlayerId = -1;
		m_aPending.Insert(request);
		request.Send(this, m_Context, "v1/health", "");

		// One timer only, whether this poll came from the interval or from an
		// accepted configuration change.
		GetGame().GetCallqueue().Remove(PollHealth_S);
		GetGame().GetCallqueue().CallLater(PollHealth_S, m_iRecheckIntervalS * 1000, false);
	}

	protected void SetState_S(TK_EBoundaryState state)
	{
		if (state == m_eState)
			return;

		Print(string.Format("[TK] backend: %1 -> %2 (contract '%3')", typename.EnumToString(TK_EBoundaryState, m_eState), typename.EnumToString(TK_EBoundaryState, state), m_sContractSeen), LogLevel.NORMAL);
		m_eState = state;
	}

	protected void HandleHealth_S(string json)
	{
		TK_HealthAnswer answer = new TK_HealthAnswer();
		answer.ExpandFromRAW(json);

		m_sContractSeen = answer.contract;
		m_iConfigRevision = answer.config_revision;
		if (answer.answer_timeout_s > 0 && answer.answer_timeout_s != m_iAnswerTimeoutS)
		{
			m_iAnswerTimeoutS = answer.answer_timeout_s;
			m_Context.SetTimeout(m_iAnswerTimeoutS);
		}
		if (answer.recheck_interval_s > 0)
			m_iRecheckIntervalS = answer.recheck_interval_s;

		if (answer.contract == TK_BackendContract.CONTRACT_VERSION)
			SetState_S(TK_EBoundaryState.READY);
		else
			SetState_S(TK_EBoundaryState.VERSION_UNKNOWN);
	}

	// =====================================================================
	// PLAYERS
	// =====================================================================

	override void OnPlayerAuditSuccess(int playerId)
	{
		super.OnPlayerAuditSuccess(playerId);

		if (!m_Context)
			return;

		// The platform identity is reliable only here and only on the server; an
		// empty id (non-dedicated play) means no record is ever created for this player.
		string uuid = GetGame().GetBackendApi().GetPlayerIdentityId(playerId);
		if (uuid == "")
		{
			Print(string.Format("[TK] backend: player %1 has no audited identity; consequential commands stay refused", playerId), LogLevel.WARNING);
			return;
		}

		TK_ConnectRequest body = new TK_ConnectRequest();
		body.display_name = GetGame().GetPlayerManager().GetPlayerName(playerId);

		TK_BackendRequest request = new TK_BackendRequest();
		request.m_eKind = TK_ERequestKind.CONNECT;
		request.m_iPlayerId = playerId;
		request.m_sTargetUuid = uuid;
		m_aPending.Insert(request);
		request.Send(this, m_Context, "v1/players/" + uuid + "/connect", body.AsString());
	}

	override void OnPlayerDisconnected(int playerId, KickCauseCode cause, int timeout)
	{
		super.OnPlayerDisconnected(playerId, cause, timeout);
		m_mSessions.Remove(playerId);
	}

	protected TK_PlayerSession GetSession(int playerId)
	{
		TK_PlayerSession session;
		m_mSessions.Find(playerId, session);
		return session;
	}

	protected int FindPlayerIdByUuid(string uuid)
	{
		foreach (int playerId, TK_PlayerSession session : m_mSessions)
		{
			if (session.m_sUuid == uuid)
				return playerId;
		}
		return -1;
	}

	// A target may be given as an identity or as the exact name of a connected
	// player; names are far easier to type at a test table.
	protected string ResolveTarget_S(int playerId, string target)
	{
		if (target == "")
		{
			TK_PlayerSession self = GetSession(playerId);
			if (self)
				return self.m_sUuid;
			return "";
		}

		PlayerManager pm = GetGame().GetPlayerManager();
		foreach (int otherId, TK_PlayerSession session : m_mSessions)
		{
			if (pm.GetPlayerName(otherId) == target)
				return session.m_sUuid;
		}
		return target;
	}

	// =====================================================================
	// REQUESTS FROM PLAYERS (called by TK_PlayerComponent on the server)
	// =====================================================================

	void SendCockpitState_S(int playerId)
	{
		TK_PlayerComponent player = TK_PlayerComponent.GetByPlayerId(playerId);
		if (!player)
			return;

		int role = 0;
		TK_PlayerSession session = GetSession(playerId);
		if (session && session.m_sRole == ROLE_OPERATOR)
			role = 1;

		string reason = "";
		if (!IsReady_S())
			reason = NotReadyReason_S();

		player.SendCockpitState_S(m_eState, reason, m_sContractSeen, m_iConfigRevision, m_iAnswerTimeoutS, m_iRecheckIntervalS, role);
	}

	// Every check the server makes before a backend call, in the order the contract
	// states. Returns "" when the request may proceed; consequential commands also
	// take the player's single pending slot.
	protected string Gate_S(int playerId, bool needOperator, bool consequential, string opId)
	{
		TK_PlayerSession session = GetSession(playerId);
		if (!session)
			return TK_Reason.IDENTITY_NOT_READY;

		if (needOperator && session.m_sRole != ROLE_OPERATOR)
		{
			ReportSecurity_S(session, "unauthorized_cockpit_rpc", "player " + playerId);
			return TK_Reason.UNAUTHORIZED;
		}

		if (session.m_sPendingOpId != "")
			return TK_Reason.BUSY;

		if (!IsReady_S() || !m_Context)
			return NotReadyReason_S();

		if (consequential)
			session.m_sPendingOpId = opId;

		return "";
	}

	protected void ReleasePending_S(int playerId, string opId)
	{
		TK_PlayerSession session = GetSession(playerId);
		if (session && session.m_sPendingOpId == opId)
			session.m_sPendingOpId = "";
	}

	void RequestWallet_S(int playerId, string target)
	{
		TK_PlayerComponent player = TK_PlayerComponent.GetByPlayerId(playerId);
		if (!player)
			return;

		string uuid = ResolveTarget_S(playerId, target);
		TK_PlayerSession session = GetSession(playerId);
		bool other = session && uuid != session.m_sUuid;

		string refusal = Gate_S(playerId, other, false, "");
		if (refusal != "")
		{
			player.SendWallet_S(target, 0, 0, 0, refusal);
			return;
		}

		TK_BackendRequest request = new TK_BackendRequest();
		request.m_eKind = TK_ERequestKind.WALLET;
		request.m_iPlayerId = playerId;
		request.m_sTargetUuid = target;
		m_aPending.Insert(request);
		request.Send(this, m_Context, "v1/players/" + uuid + "/wallet", "");
	}

	void Compensate_S(int playerId, string opId, string target, int amount, int expectedRevision, string reason)
	{
		TK_PlayerComponent player = TK_PlayerComponent.GetByPlayerId(playerId);
		if (!player)
			return;

		string refusal = Gate_S(playerId, true, true, opId);
		if (refusal != "")
		{
			player.SendCommandResult_S(opId, TK_ECommandStatus.REFUSED, refusal, 0, 0, 0);
			return;
		}

		TK_PlayerSession session = GetSession(playerId);
		TK_CompensateCommand command = new TK_CompensateCommand();
		command.op_id = opId;
		command.actor = session.m_sUuid;
		command.subject = session.m_sUuid;
		command.target = ResolveTarget_S(playerId, target);
		command.expected_revision = expectedRevision;
		command.config_revision = m_iConfigRevision;
		command.reason = reason;
		command.payload.amount = amount;

		SendCommand_S(TK_ERequestKind.COMPENSATE, playerId, opId, command.target, command.AsString());
	}

	void SetConfig_S(int playerId, string opId, int answerTimeoutS, int recheckIntervalS, string reason)
	{
		TK_PlayerComponent player = TK_PlayerComponent.GetByPlayerId(playerId);
		if (!player)
			return;

		string refusal = Gate_S(playerId, true, true, opId);
		if (refusal != "")
		{
			player.SendCommandResult_S(opId, TK_ECommandStatus.REFUSED, refusal, 0, 0, 0);
			return;
		}

		TK_PlayerSession session = GetSession(playerId);
		TK_ConfigCommand command = new TK_ConfigCommand();
		command.op_id = opId;
		command.actor = session.m_sUuid;
		command.subject = session.m_sUuid;
		command.config_revision = m_iConfigRevision;
		command.reason = reason;
		command.payload.answer_timeout_s = answerTimeoutS;
		command.payload.recheck_interval_s = recheckIntervalS;

		SendCommand_S(TK_ERequestKind.CONFIG, playerId, opId, "config", command.AsString());
	}

	void RequestAudit_S(int playerId, int beforeId)
	{
		TK_PlayerComponent player = TK_PlayerComponent.GetByPlayerId(playerId);
		if (!player)
			return;

		string refusal = Gate_S(playerId, true, false, "");
		if (refusal != "")
		{
			player.SendAuditEnd_S(0, refusal);
			return;
		}

		TK_BackendRequest request = new TK_BackendRequest();
		request.m_eKind = TK_ERequestKind.AUDIT;
		request.m_iPlayerId = playerId;
		m_aPending.Insert(request);
		request.Send(this, m_Context, string.Format("v1/audit?limit=%1&before=%2", AUDIT_PAGE, beforeId), "");
	}

	protected void SendCommand_S(TK_ERequestKind kind, int playerId, string opId, string targetUuid, string body)
	{
		TK_BackendRequest request = new TK_BackendRequest();
		request.m_eKind = kind;
		request.m_iPlayerId = playerId;
		request.m_sOpId = opId;
		request.m_sTargetUuid = targetUuid;
		m_aPending.Insert(request);
		request.Send(this, m_Context, "v1/commands", body);
	}

	// A refused cockpit RPC is put on the record by the backend under the offending
	// player's own identity. Nobody waits for the answer.
	protected void ReportSecurity_S(TK_PlayerSession session, string kind, string detail)
	{
		if (!IsReady_S() || !m_Context)
			return;

		TK_SecurityCommand command = new TK_SecurityCommand();
		command.op_id = UUID.GenV4();
		command.actor = session.m_sUuid;
		command.subject = session.m_sUuid;
		command.target = session.m_sUuid;
		command.config_revision = m_iConfigRevision;
		command.payload.kind = kind;
		command.payload.detail = detail;

		SendCommand_S(TK_ERequestKind.SECURITY, -1, command.op_id, session.m_sUuid, command.AsString());
	}

	// =====================================================================
	// ANSWERS
	// =====================================================================

	// Both callbacks land here: a 4xx may arrive through OnError with the HTTP code
	// still set, and a timeout arrives with no code at all. Anything but 200 means
	// the boundary did not answer.
	void OnRequestDone(TK_BackendRequest request, RestCallback cb)
	{
		int index = m_aPending.Find(request);
		if (index < 0)
			return;
		m_aPending.Remove(index);

		bool ok = cb.GetHttpCode() == 200;
		string json = "";
		if (ok)
			json = cb.GetData();

		switch (request.m_eKind)
		{
			case TK_ERequestKind.HEALTH:
			{
				if (ok)
					HandleHealth_S(json);
				else
					SetState_S(TK_EBoundaryState.UNREACHABLE);
				break;
			}
			case TK_ERequestKind.CONNECT:
			{
				if (ok)
					HandleConnect_S(request, json);
				else
					Print(string.Format("[TK] backend: connect for player %1 failed (HTTP %2, rest %3)", request.m_iPlayerId, cb.GetHttpCode(), cb.GetRestResult()), LogLevel.WARNING);
				break;
			}
			case TK_ERequestKind.WALLET:
			{
				HandleWallet_S(request, ok, json);
				break;
			}
			case TK_ERequestKind.COMPENSATE:
			case TK_ERequestKind.CONFIG:
			{
				ReleasePending_S(request.m_iPlayerId, request.m_sOpId);
				HandleCommand_S(request, ok, json);
				break;
			}
			case TK_ERequestKind.SECURITY:
			{
				break;
			}
			case TK_ERequestKind.AUDIT:
			{
				HandleAudit_S(request, ok, json);
				break;
			}
		}
	}

	protected void HandleConnect_S(TK_BackendRequest request, string json)
	{
		TK_ConnectAnswer answer = new TK_ConnectAnswer();
		answer.ExpandFromRAW(json);

		TK_PlayerSession session = new TK_PlayerSession();
		session.m_sUuid = request.m_sTargetUuid;
		session.m_sRole = answer.player.role;
		m_mSessions.Set(request.m_iPlayerId, session);

		TK_PlayerComponent player = TK_PlayerComponent.GetByPlayerId(request.m_iPlayerId);
		if (!player)
			return;

		foreach (TK_ReceiptData receipt : answer.receipts)
			player.SendReceipt_S(receipt.op_id, receipt.amount, receipt.total_after, receipt.created_at, receipt.reason);
	}

	protected void HandleWallet_S(TK_BackendRequest request, bool ok, string json)
	{
		TK_PlayerComponent player = TK_PlayerComponent.GetByPlayerId(request.m_iPlayerId);
		if (!player)
			return;

		if (!ok)
		{
			player.SendWallet_S(request.m_sTargetUuid, 0, 0, 0, TK_Reason.BACKEND_UNREACHABLE);
			return;
		}

		TK_WalletData wallet = new TK_WalletData();
		wallet.ExpandFromRAW(json);
		player.SendWallet_S(request.m_sTargetUuid, wallet.total, wallet.reserved, wallet.revision, wallet.error);
	}

	protected void HandleCommand_S(TK_BackendRequest request, bool ok, string json)
	{
		TK_PlayerComponent player = TK_PlayerComponent.GetByPlayerId(request.m_iPlayerId);

		if (!ok)
		{
			if (player)
				player.SendCommandResult_S(request.m_sOpId, TK_ECommandStatus.REFUSED, TK_Reason.BACKEND_UNREACHABLE, 0, 0, 0);
			return;
		}

		TK_CommandAnswer answer = new TK_CommandAnswer();
		answer.ExpandFromRAW(json);

		TK_ECommandStatus status = TK_ECommandStatus.REFUSED;
		if (answer.status == "accepted")
			status = TK_ECommandStatus.ACCEPTED;
		else if (answer.status == "already_applied")
			status = TK_ECommandStatus.ALREADY_APPLIED;

		if (player)
			player.SendCommandResult_S(request.m_sOpId, status, answer.reason_code, answer.wallet.total, answer.wallet.reserved, answer.revision);

		if (status == TK_ECommandStatus.ACCEPTED && request.m_eKind == TK_ERequestKind.COMPENSATE)
		{
			int targetId = FindPlayerIdByUuid(request.m_sTargetUuid);
			TK_PlayerComponent target = TK_PlayerComponent.GetByPlayerId(targetId);
			if (target)
				target.SendReceipt_S(answer.receipt.op_id, answer.receipt.amount, answer.receipt.total_after, answer.receipt.created_at, answer.receipt.reason);
		}

		// The new runtime values arrive with the next health answer; ask now rather
		// than wait out the old interval.
		if (status == TK_ECommandStatus.ACCEPTED && request.m_eKind == TK_ERequestKind.CONFIG)
			PollHealth_S();
	}

	protected void HandleAudit_S(TK_BackendRequest request, bool ok, string json)
	{
		TK_PlayerComponent player = TK_PlayerComponent.GetByPlayerId(request.m_iPlayerId);
		if (!player)
			return;

		if (!ok)
		{
			player.SendAuditEnd_S(0, TK_Reason.BACKEND_UNREACHABLE);
			return;
		}

		TK_AuditAnswer answer = new TK_AuditAnswer();
		answer.ExpandFromRAW(json);

		player.SendAuditBegin_S();
		int lastId = 0;
		foreach (TK_AuditRowData row : answer.entries)
		{
			player.SendAuditRow_S(row.id, row.created_at, row.actor_uuid, row.type, row.target, row.outcome, row.reason_code, row.amount);
			lastId = row.id;
		}
		player.SendAuditEnd_S(lastId, "");
	}
}
