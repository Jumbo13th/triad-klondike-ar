// Contract version 1 with the local backend (specs/001-backend-boundary/contracts/
// backend-http.md). JsonApiStruct ignores keys it does not register, so the backend
// may add fields without breaking a build that still speaks this version.

class TK_BackendContract
{
	static const string CONTRACT_VERSION = "1";
}

// Refusal codes the client translates as #TK-Reason_<code>. The first four are
// decided on the game server before any backend call.
class TK_Reason
{
	static const string BACKEND_UNREACHABLE = "backend_unreachable";
	static const string CONTRACT_VERSION_UNKNOWN = "contract_version_unknown";
	static const string IDENTITY_NOT_READY = "identity_not_ready";
	static const string BUSY = "busy";
	static const string UNAUTHORIZED = "unauthorized";
	static const string PLAYER_UNKNOWN = "player_unknown";
}

// Outcome of a command as the owner RPCs carry it.
enum TK_ECommandStatus
{
	ACCEPTED,
	REFUSED,
	ALREADY_APPLIED,
}

class TK_HealthAnswer : JsonApiStruct
{
	string contract;
	int config_revision;
	int answer_timeout_s;
	int recheck_interval_s;

	void TK_HealthAnswer()
	{
		RegV("contract");
		RegV("config_revision");
		RegV("answer_timeout_s");
		RegV("recheck_interval_s");
	}
}

class TK_WalletData : JsonApiStruct
{
	int total;
	int reserved;
	int revision;
	string error;

	void TK_WalletData()
	{
		RegV("total");
		RegV("reserved");
		RegV("revision");
		RegV("error");
	}
}

class TK_ReceiptData : JsonApiStruct
{
	string op_id;
	string kind;
	int amount;
	int total_after;
	string reason;
	string created_at;

	void TK_ReceiptData()
	{
		RegV("op_id");
		RegV("kind");
		RegV("amount");
		RegV("total_after");
		RegV("reason");
		RegV("created_at");
	}
}

class TK_PlayerData : JsonApiStruct
{
	string uuid;
	string role;

	void TK_PlayerData()
	{
		RegV("uuid");
		RegV("role");
	}
}

class TK_ConnectRequest : JsonApiStruct
{
	string display_name;

	void TK_ConnectRequest()
	{
		RegV("display_name");
	}
}

class TK_ConnectAnswer : JsonApiStruct
{
	ref TK_PlayerData player = new TK_PlayerData();
	ref TK_WalletData wallet = new TK_WalletData();
	ref array<ref TK_ReceiptData> receipts = {};

	void TK_ConnectAnswer()
	{
		RegV("player");
		RegV("wallet");
		RegV("receipts");
	}
}

// Envelope fields shared by every command; each command type adds its payload.
class TK_CommandBase : JsonApiStruct
{
	string op_id;
	string type;
	string actor;
	string subject;
	string target;
	int expected_revision;
	int config_revision;
	string reason;
	// True when the target has a live session, so the receipt is pushed at once and
	// the backend need not replay it on the next connect.
	bool target_online;

	void TK_CommandBase()
	{
		RegV("op_id");
		RegV("type");
		RegV("actor");
		RegV("subject");
		RegV("target");
		RegV("expected_revision");
		RegV("config_revision");
		RegV("reason");
		RegV("target_online");
	}
}

class TK_CompensatePayload : JsonApiStruct
{
	int amount;

	void TK_CompensatePayload()
	{
		RegV("amount");
	}
}

class TK_CompensateCommand : TK_CommandBase
{
	ref TK_CompensatePayload payload = new TK_CompensatePayload();

	void TK_CompensateCommand()
	{
		type = "wallet.compensate";
		RegV("payload");
	}
}

class TK_ConfigPayload : JsonApiStruct
{
	int answer_timeout_s;
	int recheck_interval_s;

	void TK_ConfigPayload()
	{
		RegV("answer_timeout_s");
		RegV("recheck_interval_s");
	}
}

class TK_ConfigCommand : TK_CommandBase
{
	ref TK_ConfigPayload payload = new TK_ConfigPayload();

	void TK_ConfigCommand()
	{
		type = "config.set";
		target = "config";
		RegV("payload");
	}
}

class TK_SecurityPayload : JsonApiStruct
{
	string kind;
	string detail;

	void TK_SecurityPayload()
	{
		RegV("kind");
		RegV("detail");
	}
}

class TK_SecurityCommand : TK_CommandBase
{
	ref TK_SecurityPayload payload = new TK_SecurityPayload();

	void TK_SecurityCommand()
	{
		type = "security";
		RegV("payload");
	}
}

class TK_CommandAnswer : JsonApiStruct
{
	string status;
	string reason_code;
	int revision;
	ref TK_WalletData wallet = new TK_WalletData();
	ref TK_ReceiptData receipt = new TK_ReceiptData();

	void TK_CommandAnswer()
	{
		RegV("status");
		RegV("reason_code");
		RegV("revision");
		RegV("wallet");
		RegV("receipt");
	}
}

class TK_AuditRowData : JsonApiStruct
{
	int id;
	string created_at;
	string op_id;
	string actor_uuid;
	string type;
	string target;
	string outcome;
	string reason_code;
	int amount;

	void TK_AuditRowData()
	{
		RegV("id");
		RegV("created_at");
		RegV("op_id");
		RegV("actor_uuid");
		RegV("type");
		RegV("target");
		RegV("outcome");
		RegV("reason_code");
		RegV("amount");
	}
}

class TK_AuditAnswer : JsonApiStruct
{
	ref array<ref TK_AuditRowData> entries = {};

	void TK_AuditAnswer()
	{
		RegV("entries");
	}
}
