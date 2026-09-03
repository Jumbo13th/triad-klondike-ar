class TK_GameModeClass : SCR_BaseGameModeClass
{
}

// Klondike game mode. Gameplay data lives in the game-mode components; this class
// only owns what the base game mode requires and the cockpit key.
class TK_GameMode : SCR_BaseGameMode
{
	override void OnGameStart()
	{
		super.OnGameStart();

		// Dedicated servers have no input and no UI.
		if (RplSession.Mode() != RplMode.Dedicated)
			GetGame().GetInputManager().AddActionListener("TK_OpenCockpit", EActionTrigger.DOWN, Action_OpenCockpit);
	}

	protected void Action_OpenCockpit()
	{
		TK_CockpitMenu.Open();
	}
}
