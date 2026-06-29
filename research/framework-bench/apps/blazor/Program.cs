using Microsoft.AspNetCore.Components.WebAssembly.Hosting;
using BillSplitter;
using BillSplitter.Services;

var builder = WebAssemblyHostBuilder.CreateDefault(args);
builder.RootComponents.Add<SplitterApp>("#app");
builder.Services.AddSingleton<SettingsService>();
await builder.Build().RunAsync();
