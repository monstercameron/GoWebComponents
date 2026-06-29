namespace BillSplitter.Services;

// Shared/global state as a DI singleton with a change notification — the
// idiomatic Blazor cross-component shared-state pattern. Components subscribe to
// OnChange and call StateHasChanged.
public class SettingsService
{
    public string Theme { get; private set; } = "light";
    public bool RoundUp { get; private set; }

    public event Action? OnChange;

    public void ToggleTheme()
    {
        Theme = Theme == "dark" ? "light" : "dark";
        OnChange?.Invoke();
    }

    public void ToggleRoundUp()
    {
        RoundUp = !RoundUp;
        OnChange?.Invoke();
    }
}
