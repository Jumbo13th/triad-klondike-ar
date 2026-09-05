class TK_GameModeClass : SCR_BaseGameModeClass
{
}

// Klondike game mode. Gameplay data lives in the game-mode components; this class
// only owns what the base game mode requires and the cockpit key.
class TK_GameMode : SCR_BaseGameMode
{
	// The cockpit key is bound everywhere but on a dedicated server, which has no
	// input and no UI.
	override void OnGameStart()
	{
		super.OnGameStart();

		if (RplSession.Mode() != RplMode.Dedicated)
			GetGame().GetInputManager().AddActionListener("TK_OpenCockpit", EActionTrigger.DOWN, Action_OpenCockpit);
	}

	protected void Action_OpenCockpit()
	{
		TK_CockpitMenu.Open();
	}
}
