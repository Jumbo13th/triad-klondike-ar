modded enum ChimeraMenuPreset : ScriptMenuPresetEnum
{
	TK_CockpitMenu,
}

// The first slice of the launch cockpit. Everyone sees the backend state and their
// own balance; the operator section (wallet lookup, compensation, runtime values,
// audit) is shown only when the server says the player holds the operator role. The
// panel asks the server on open and on refresh; only receipts arrive unrequested.
class TK_CockpitMenu : MenuBase
{
	protected static const ResourceName LINE_LAYOUT = "{9494DF8DFD0569EE}UI/Cockpit/CockpitTextLine.layout";

	protected TK_PlayerComponent m_Player;
	protected Widget m_wRoot;
	protected TextWidget m_wStateText;
	protected TextWidget m_wContractText;
	protected TextWidget m_wConfigText;
	protected TextWidget m_wBalanceText;
	protected TextWidget m_wReceiptText;
	protected Widget m_wOperatorSection;
	protected EditBoxWidget m_wWalletEdit;
	protected TextWidget m_wWalletResult;
	protected EditBoxWidget m_wAmountEdit;
	protected EditBoxWidget m_wReasonEdit;
	protected TextWidget m_wResultText;
	protected EditBoxWidget m_wTimeoutEdit;
	protected EditBoxWidget m_wIntervalEdit;
	protected Widget m_wAuditList;
	protected Widget m_wMoreButton;

	protected bool m_bOperator;

	// The operation id survives a "backend unreachable" answer so Retry resends the
	// same command; any other terminal answer clears it.
	protected string m_sPendingOpId;
	protected bool m_bPendingIsConfig;
	// While an answer is outstanding a new press must not replace the pending operation,
	// or the answer that arrives for it is dropped as unknown.
	protected bool m_bAwaitingAnswer;
	protected string m_sPendingTarget;
	protected int m_iPendingAmount;
	protected int m_iPendingTimeout;
	protected int m_iPendingInterval;
	protected string m_sPendingReason;

	protected string m_sLookupTarget;
	protected int m_iLookupRevision;
	protected int m_iAuditNextBefore;
	protected bool m_bAuditAppend;

	static void Open()
	{
		MenuManager menuManager = GetGame().GetMenuManager();
		if (!menuManager)
			return;

		if (menuManager.FindMenuByPreset(ChimeraMenuPreset.TK_CockpitMenu))
			return;

		menuManager.OpenMenu(ChimeraMenuPreset.TK_CockpitMenu);
	}

	override void OnMenuOpen()
	{
		if (RplSession.Mode() == RplMode.Dedicated)
		{
			Close();
			return;
		}

		m_wRoot = GetRootWidget();
		m_Player = TK_PlayerComponent.GetLocalInstance();
		if (!m_wRoot || !m_Player)
		{
			Print("[TK] cockpit opened without a root widget or a local player component", LogLevel.WARNING);
			Close();
			return;
		}

		m_wStateText = TextWidget.Cast(m_wRoot.FindAnyWidget("StateText"));
		m_wContractText = TextWidget.Cast(m_wRoot.FindAnyWidget("ContractText"));
		m_wConfigText = TextWidget.Cast(m_wRoot.FindAnyWidget("ConfigText"));
		m_wBalanceText = TextWidget.Cast(m_wRoot.FindAnyWidget("PlayerBalanceText"));
		m_wReceiptText = TextWidget.Cast(m_wRoot.FindAnyWidget("ReceiptText"));
		m_wOperatorSection = m_wRoot.FindAnyWidget("OperatorSection");
		m_wWalletEdit = EditBoxWidget.Cast(m_wRoot.FindAnyWidget("WalletUuidEdit"));
		m_wWalletResult = TextWidget.Cast(m_wRoot.FindAnyWidget("WalletResultText"));
		m_wAmountEdit = EditBoxWidget.Cast(m_wRoot.FindAnyWidget("AmountEdit"));
		m_wReasonEdit = EditBoxWidget.Cast(m_wRoot.FindAnyWidget("ReasonEdit"));
		m_wResultText = TextWidget.Cast(m_wRoot.FindAnyWidget("ResultText"));
		m_wTimeoutEdit = EditBoxWidget.Cast(m_wRoot.FindAnyWidget("TimeoutEdit"));
		m_wIntervalEdit = EditBoxWidget.Cast(m_wRoot.FindAnyWidget("IntervalEdit"));
		m_wAuditList = m_wRoot.FindAnyWidget("AuditList");
		m_wMoreButton = m_wRoot.FindAnyWidget("AuditMoreButton");

		SCR_ButtonBaseComponent button = FindButton("LookupButton");
		if (button)
			button.m_OnClicked.Insert(OnLookup);
		button = FindButton("CreditButton");
		if (button)
			button.m_OnClicked.Insert(OnApply);
		button = FindButton("RetryButton");
		if (button)
			button.m_OnClicked.Insert(OnRetry);
		button = FindButton("ApplyConfigButton");
		if (button)
			button.m_OnClicked.Insert(OnApplyConfig);
		button = FindButton("AuditMoreButton");
		if (button)
			button.m_OnClicked.Insert(OnAuditMore);
		button = FindButton("RefreshButton");
		if (button)
			button.m_OnClicked.Insert(Refresh);
		button = FindButton("CloseButton");
		if (button)
			button.m_OnClicked.Insert(OnClose);

		GetGame().GetInputManager().AddActionListener("MenuBack", EActionTrigger.DOWN, OnBackAction);

		m_Player.m_OnCockpitState.Insert(OnCockpitState);
		m_Player.m_OnWallet.Insert(OnWallet);
		m_Player.m_OnCommandResult.Insert(OnCommandResult);
		m_Player.m_OnReceipt.Insert(OnReceipt);
		m_Player.m_OnAuditBegin.Insert(OnAuditBegin);
		m_Player.m_OnAuditRow.Insert(OnAuditRow);
		m_Player.m_OnAuditEnd.Insert(OnAuditEnd);

		if (m_wOperatorSection)
			m_wOperatorSection.SetVisible(false);
		if (m_wMoreButton)
			m_wMoreButton.SetVisible(false);

		Refresh();
	}

	override void OnMenuClose()
	{
		GetGame().GetInputManager().RemoveActionListener("MenuBack", EActionTrigger.DOWN, OnBackAction);

		if (!m_Player)
			return;

		m_Player.m_OnCockpitState.Remove(OnCockpitState);
		m_Player.m_OnWallet.Remove(OnWallet);
		m_Player.m_OnCommandResult.Remove(OnCommandResult);
		m_Player.m_OnReceipt.Remove(OnReceipt);
		m_Player.m_OnAuditBegin.Remove(OnAuditBegin);
		m_Player.m_OnAuditRow.Remove(OnAuditRow);
		m_Player.m_OnAuditEnd.Remove(OnAuditEnd);
	}

	protected SCR_ButtonBaseComponent FindButton(string name)
	{
		Widget button = m_wRoot.FindAnyWidget(name);
		if (!button)
			return null;

		return SCR_ButtonBaseComponent.Cast(button.FindHandler(SCR_ButtonBaseComponent));
	}

	protected void OnClose()
	{
		Close();
	}

	protected void OnBackAction()
	{
		if (!IsFocused())
			return;

		Close();
	}

	protected void Refresh()
	{
		m_Player.AskCockpitState();
		m_Player.AskWallet("");
		if (m_bOperator)
		{
			m_bAuditAppend = false;
			m_Player.AskAudit(0);
		}
	}

	// =====================================================================
	// STATE AND BALANCE
	// =====================================================================

	// The configuration edit boxes are prefilled only while empty, so a refresh never
	// overwrites what the operator has typed.
	protected void OnCockpitState(int state, string reasonCode, string contractSeen, int configRevision, int answerTimeoutS, int recheckIntervalS, int role)
	{
		string stateText = WidgetManager.Translate(StateKey(state));
		if (reasonCode != "")
			stateText += " (" + WidgetManager.Translate("#TK-Reason_" + reasonCode) + ")";
		if (m_wStateText)
			m_wStateText.SetText(WidgetManager.Translate("#TK-Cockpit_Boundary") + ": " + stateText);
		if (m_wContractText)
			m_wContractText.SetTextFormat("#TK-Cockpit_Contract", contractSeen, configRevision);
		if (m_wConfigText)
			m_wConfigText.SetTextFormat("#TK-Cockpit_Config", answerTimeoutS, recheckIntervalS);

		bool wasOperator = m_bOperator;
		m_bOperator = role == 1;
		if (m_wOperatorSection)
			m_wOperatorSection.SetVisible(m_bOperator);
		if (m_wMoreButton)
			m_wMoreButton.SetVisible(m_bOperator);

		if (m_bOperator)
		{
			if (m_wTimeoutEdit && m_wTimeoutEdit.GetText() == "")
				m_wTimeoutEdit.SetText(answerTimeoutS.ToString());
			if (m_wIntervalEdit && m_wIntervalEdit.GetText() == "")
				m_wIntervalEdit.SetText(recheckIntervalS.ToString());
		}

		if (m_bOperator && !wasOperator)
		{
			m_bAuditAppend = false;
			m_Player.AskAudit(0);
		}
	}

	protected static string StateKey(int state)
	{
		switch (state)
		{
			case TK_EBoundaryState.READY: return "#TK-State_Ready";
			case TK_EBoundaryState.UNREACHABLE: return "#TK-State_Unreachable";
			case TK_EBoundaryState.VERSION_UNKNOWN: return "#TK-State_VersionUnknown";
		}
		return "#TK-State_NotChecked";
	}

	protected void OnWallet(string target, int total, int reserved, int revision, string reasonCode)
	{
		if (target == "")
		{
			if (!m_wBalanceText)
				return;
			if (reasonCode != "")
				m_wBalanceText.SetTextFormat("#TK-Cockpit_BalanceUnknown", WidgetManager.Translate("#TK-Reason_" + reasonCode));
			else
				m_wBalanceText.SetTextFormat("#TK-Cockpit_YourBalance", total, reserved);
			return;
		}

		if (!m_wWalletResult)
			return;
		if (reasonCode != "")
		{
			m_wWalletResult.SetTextFormat("#TK-Cockpit_Refused", WidgetManager.Translate("#TK-Reason_" + reasonCode));
			return;
		}
		m_sLookupTarget = target;
		m_iLookupRevision = revision;
		m_wWalletResult.SetTextFormat("#TK-Cockpit_WalletResult", total, reserved, revision);
	}

	protected void OnReceipt(string opId, int amount, int totalAfter, string time, string reason)
	{
		if (m_wReceiptText)
			m_wReceiptText.SetTextFormat("#TK-Receipt_Compensation", amount, totalAfter, reason);
		if (m_wBalanceText)
			m_wBalanceText.SetTextFormat("#TK-Cockpit_YourBalance", totalAfter, 0);
	}

	// =====================================================================
	// OPERATOR COMMANDS
	// =====================================================================

	protected void OnLookup()
	{
		if (!m_wWalletEdit)
			return;

		string target = m_wWalletEdit.GetText();
		target.TrimInPlace();
		if (target == "")
			return;

		m_Player.AskWallet(target);
	}

	protected void OnApply()
	{
		if (!m_wAmountEdit || !m_wReasonEdit || !m_wWalletEdit)
			return;
		if (RefuseWhileAwaiting())
			return;

		string amountText = m_wAmountEdit.GetText();
		amountText.TrimInPlace();
		int amount = amountText.ToInt();
		if (amount == 0)
		{
			ShowResult(WidgetManager.Translate("#TK-Cockpit_AmountInvalid"));
			return;
		}

		string reason = m_wReasonEdit.GetText();
		reason.TrimInPlace();
		if (reason == "")
		{
			ShowResult(WidgetManager.Translate("#TK-Cockpit_Refused", WidgetManager.Translate("#TK-Reason_reason_required")));
			return;
		}

		string target = m_wWalletEdit.GetText();
		target.TrimInPlace();
		if (target != m_sLookupTarget)
			m_iLookupRevision = 0;

		m_sPendingOpId = UUID.GenV4();
		m_bPendingIsConfig = false;
		m_sPendingTarget = target;
		m_iPendingAmount = amount;
		m_sPendingReason = reason;
		SendPending();
	}

	protected void OnApplyConfig()
	{
		if (!m_wTimeoutEdit || !m_wIntervalEdit || !m_wReasonEdit)
			return;
		if (RefuseWhileAwaiting())
			return;

		string reason = m_wReasonEdit.GetText();
		reason.TrimInPlace();
		if (reason == "")
		{
			ShowResult(WidgetManager.Translate("#TK-Cockpit_Refused", WidgetManager.Translate("#TK-Reason_reason_required")));
			return;
		}

		m_sPendingOpId = UUID.GenV4();
		m_bPendingIsConfig = true;
		m_iPendingTimeout = m_wTimeoutEdit.GetText().ToInt();
		m_iPendingInterval = m_wIntervalEdit.GetText().ToInt();
		m_sPendingReason = reason;
		SendPending();
	}

	protected void OnRetry()
	{
		if (m_sPendingOpId == "")
			return;
		if (RefuseWhileAwaiting())
			return;

		SendPending();
	}

	protected bool RefuseWhileAwaiting()
	{
		if (!m_bAwaitingAnswer)
			return false;
		ShowResult(WidgetManager.Translate("#TK-Cockpit_Refused", WidgetManager.Translate("#TK-Reason_busy")));
		return true;
	}

	protected void SendPending()
	{
		m_bAwaitingAnswer = true;
		ShowResult(WidgetManager.Translate("#TK-Cockpit_Pending"));
		if (m_bPendingIsConfig)
			m_Player.AskSetConfig(m_sPendingOpId, m_iPendingTimeout, m_iPendingInterval, m_sPendingReason);
		else
			m_Player.AskCompensate(m_sPendingOpId, m_sPendingTarget, m_iPendingAmount, m_iLookupRevision, m_sPendingReason);
	}

	// A stale revision comes back with the current one; it counts as the lookup of the
	// pending target, or the next press would reset it to zero again.
	protected void OnCommandResult(string opId, int status, string reasonCode, int total, int reserved, int revision)
	{
		if (opId != m_sPendingOpId)
			return;
		m_bAwaitingAnswer = false;

		switch (status)
		{
			case TK_ECommandStatus.ACCEPTED:
			{
				if (m_bPendingIsConfig)
				{
					ShowResult(WidgetManager.Translate("#TK-Cockpit_ConfigAccepted", revision));
					m_Player.AskCockpitState();
				}
				else
				{
					ShowResult(WidgetManager.Translate("#TK-Cockpit_Accepted", total, revision));
					m_sLookupTarget = m_sPendingTarget;
					m_iLookupRevision = revision;
					if (m_wWalletResult)
						m_wWalletResult.SetTextFormat("#TK-Cockpit_WalletResult", total, reserved, revision);
				}
				m_sPendingOpId = "";
				m_bAuditAppend = false;
				m_Player.AskAudit(0);
				break;
			}
			case TK_ECommandStatus.ALREADY_APPLIED:
			{
				ShowResult(WidgetManager.Translate("#TK-Cockpit_AlreadyApplied", total, revision));
				if (!m_bPendingIsConfig)
				{
					m_sLookupTarget = m_sPendingTarget;
					m_iLookupRevision = revision;
				}
				m_sPendingOpId = "";
				break;
			}
			default:
			{
				ShowResult(WidgetManager.Translate("#TK-Cockpit_Refused", WidgetManager.Translate("#TK-Reason_" + reasonCode)));
				if (revision > 0 && !m_bPendingIsConfig)
				{
					m_sLookupTarget = m_sPendingTarget;
					m_iLookupRevision = revision;
				}
				if (reasonCode != TK_Reason.BACKEND_UNREACHABLE)
					m_sPendingOpId = "";
				break;
			}
		}
	}

	protected void ShowResult(string text)
	{
		if (m_wResultText)
			m_wResultText.SetText(text);
	}

	// =====================================================================
	// AUDIT
	// =====================================================================

	protected void OnAuditMore()
	{
		if (m_iAuditNextBefore <= 0)
			return;

		m_bAuditAppend = true;
		m_Player.AskAudit(m_iAuditNextBefore);
	}

	protected void OnAuditBegin()
	{
		if (!m_wAuditList || m_bAuditAppend)
			return;

		Widget child = m_wAuditList.GetChildren();
		while (child)
		{
			Widget next = child.GetSibling();
			m_wAuditList.RemoveChild(child);
			child = next;
		}
	}

	protected void OnAuditRow(int id, string time, string actorUuid, string type, string target, string outcome, string reasonCode, int amount)
	{
		string detail = WidgetManager.Translate("#TK-Outcome_" + outcome);
		if (reasonCode != "")
			detail += " " + WidgetManager.Translate("#TK-Reason_" + reasonCode);
		AddLine(WidgetManager.Translate("#TK-Cockpit_AuditRow", time, type, detail, target, amount));
	}

	protected void OnAuditEnd(int nextBeforeId, string reasonCode)
	{
		m_iAuditNextBefore = nextBeforeId;
		if (reasonCode != "")
		{
			AddLine(WidgetManager.Translate("#TK-Cockpit_Refused", WidgetManager.Translate("#TK-Reason_" + reasonCode)));
			return;
		}
		if (m_wAuditList && !m_wAuditList.GetChildren())
			AddLine(WidgetManager.Translate("#TK-Cockpit_NoAudit"));
	}

	protected void AddLine(string text)
	{
		if (!m_wAuditList)
			return;

		TextWidget line = TextWidget.Cast(GetGame().GetWorkspace().CreateWidgets(LINE_LAYOUT, m_wAuditList));
		if (line)
			line.SetText(text);
	}
}
