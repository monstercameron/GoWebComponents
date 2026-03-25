//go:build js && wasm

package app

import (
	"strings"

	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/ui"
)

const (
	chatI18nNamespace        = "chat"
	chatLocalePersistenceKey = "chat-wizard:locale"
)

type localeOption struct {
	ID          string
	NativeLabel string
}

var availableLocales = []localeOption{
	{ID: "en", NativeLabel: "English"},
	{ID: "es", NativeLabel: "Español"},
	{ID: "fr", NativeLabel: "Français"},
}

var chatWizardBundle = func() *i18n.Bundle {
	bundle := i18n.NewBundle(i18n.BundleOptions{DefaultLocale: "en", FallbackLocale: "en"})
	bundle.Register("en", i18n.Catalog{
		chatI18nNamespace: {
			"auth.badge":                         {Text: "GWC auth"},
			"auth.connecting":                    {Text: "Connecting to the local gRPC bridge..."},
			"auth.resolving":                     {Text: "Resolving the saved session..."},
			"auth.heroTitle":                     {Text: "Run the chat experiment without a separate login site."},
			"auth.heroBody":                      {Text: "The root route now stays inside the Go/WASM app. Sign in or create an account over gRPC, then keep everything in one shell."},
			"auth.grpcBadge":                     {Text: "gRPC only"},
			"auth.grpcBody":                      {Text: "HTTP only serves the shell and static assets. Account actions now go through the chat service tunnel."},
			"auth.loginTitle":                    {Text: "Sign in"},
			"auth.loginBody":                     {Text: "Use your local experiment account to continue the current workspace."},
			"auth.signupTitle":                   {Text: "Create account"},
			"auth.signupBody":                    {Text: "Create a local account for this experiment. No separate REST auth pages."},
			"auth.signIn":                        {Text: "Sign in"},
			"auth.createAccount":                 {Text: "Create account"},
			"auth.switchToSignup":                {Text: "Need an account? Create one"},
			"auth.switchToLogin":                 {Text: "Already have an account? Sign in"},
			"auth.displayName":                   {Text: "Display name"},
			"auth.displayNamePlaceholder":        {Text: "Optional display name"},
			"auth.email":                         {Text: "Email"},
			"auth.emailPlaceholder":              {Text: "you@example.com"},
			"auth.password":                      {Text: "Password"},
			"auth.passwordPlaceholder":           {Text: "Enter a password"},
			"auth.passwordHelp":                  {Text: "Use at least 8 characters for sign up."},
			"auth.logout":                        {Text: "Sign out"},
			"empty.heading":                      {Text: "Explore the Go WASM UI experiment"},
			"empty.body":                         {Text: "Ask about Go, Golang, WebAssembly, UI patterns, or your next framework experiment."},
			"input.placeholder":                  {Text: "Message the GoWebComponents experiment..."},
			"input.disclaimer":                   {Text: "Experimental Go/WASM UI output can be wrong. Verify important details."},
			"input.threadTotal":                  {Text: "Thread total {cost}"},
			"input.threadTotalPartial":           {Text: "Thread total {cost} (partial)"},
			"controls.provider":                  {Text: "Provider"},
			"controls.model":                     {Text: "Model"},
			"controls.intelligence":              {Text: "Intelligence"},
			"sidebar.newChat":                    {Text: "New chat"},
			"sidebar.noConversations":            {Text: "No conversations yet"},
			"sidebar.emptyConversation":          {Text: "Empty conversation"},
			"sidebar.editSettings":               {Text: "Edit settings"},
			"sidebar.deleteConversation":         {Text: "Delete conversation"},
			"quote.prompt":                       {Text: "Quote text"},
			"canvas.badge":                       {Text: "Canvas"},
			"canvas.title":                       {Text: "Live preview"},
			"canvas.caption":                     {Text: "Latest ```canvas block"},
			"message.thinking":                   {Text: "Thinking"},
			"message.live":                       {Text: "Live"},
			"message.switchedTo":                 {Text: "switched to {model}"},
			"message.cancel":                     {Text: "Cancel"},
			"message.saveResend":                 {Text: "Save and resend"},
			"message.copy":                       {Text: "Copy"},
			"message.edit":                       {Text: "Edit"},
			"message.fork":                       {Text: "Fork"},
			"assistant.loading":                  {Text: "Loading"},
			"assistant.pause":                    {Text: "Pause"},
			"assistant.play":                     {Text: "Play"},
			"assistant.stop":                     {Text: "Stop"},
			"assistant.speechUnavailable":        {Text: "Speech unavailable"},
			"modal.deleteTitle":                  {Text: "Delete conversation?"},
			"modal.deleteBody":                   {Text: "This will permanently delete this chat and all its messages. This cannot be undone."},
			"modal.settingsTitle":                {Text: "Settings"},
			"modal.displayName":                  {Text: "Display name"},
			"modal.displayNamePlaceholder":       {Text: "Enter your name..."},
			"modal.aiTone":                       {Text: "AI tone"},
			"modal.intelligence":                 {Text: "Reasoning"},
			"modal.intelligenceHelp":             {Text: "Choose how much reasoning the assistant should use before it answers."},
			"modal.intelligenceUnavailable":      {Text: "The currently selected model does not expose reasoning controls."},
			"modal.systemPrompt":                 {Text: "System prompt"},
			"modal.systemPromptPlaceholder":      {Text: "Add durable instructions for how the assistant should respond..."},
			"modal.systemPromptHelp":             {Text: "These instructions are added on top of the built-in assistant prompt for your account."},
			"modal.memories":                     {Text: "Remembered preferences"},
			"modal.memoriesHelp":                 {Text: "Review and edit the profile details the assistant remembers about you."},
			"modal.memoriesEmpty":                {Text: "No remembered preferences yet."},
			"modal.memoryAdd":                    {Text: "Add memory"},
			"modal.memoryItem":                   {Text: "Profile item"},
			"modal.memoryDelete":                 {Text: "Delete"},
			"modal.memoryCategoryPlaceholder":    {Text: "Category, for example preference or project"},
			"modal.memorySummaryPlaceholder":     {Text: "Short summary"},
			"modal.memoryDetailPlaceholder":      {Text: "Optional detail"},
			"modal.memoryReasonPlaceholder":      {Text: "Why this matters"},
			"modal.language":                     {Text: "Language"},
			"modal.save":                         {Text: "Save"},
			"tone.balanced.label":                {Text: "Balanced"},
			"tone.balanced.desc":                 {Text: "Helpful and conversational"},
			"tone.professional.label":            {Text: "Professional"},
			"tone.professional.desc":             {Text: "Formal and precise"},
			"tone.concise.label":                 {Text: "Concise"},
			"tone.concise.desc":                  {Text: "Brief and to the point"},
			"thinking.off":                       {Text: "Off"},
			"thinking.low":                       {Text: "Low"},
			"thinking.medium":                    {Text: "Med"},
			"thinking.high":                      {Text: "Hi"},
			"error.genericPrefix":                {Text: "ERROR"},
			"error.receivePrefix":                {Text: "ERROR recv:"},
			"tts.audioPlaybackFailed":            {Text: "Audio playback failed"},
			"tts.audioUnavailable":               {Text: "Audio output is unavailable in this browser"},
			"tts.audioCouldNotStart":             {Text: "Audio playback could not start"},
			"tts.audioCouldNotResume":            {Text: "Audio playback could not resume"},
			"tts.noText":                         {Text: "Nothing to read aloud"},
			"tts.connectionNotReady":             {Text: "Chat connection is not ready yet"},
			"tts.speechSynthesisFailed":          {Text: "Speech synthesis failed"},
			"tts.speechSynthesisReturnedNoAudio": {Text: "Speech synthesis returned no audio"},
		},
	})
	bundle.Register("es", i18n.Catalog{
		chatI18nNamespace: {
			"auth.badge":                         {Text: "Auth GWC"},
			"auth.connecting":                    {Text: "Conectando con el puente gRPC local..."},
			"auth.resolving":                     {Text: "Resolviendo la sesion guardada..."},
			"auth.heroTitle":                     {Text: "Ejecuta el experimento de chat sin un sitio de acceso separado."},
			"auth.heroBody":                      {Text: "La ruta raiz ahora se mantiene dentro de la app Go/WASM. Inicia sesion o crea una cuenta por gRPC y conserva todo en una sola interfaz."},
			"auth.grpcBadge":                     {Text: "Solo gRPC"},
			"auth.grpcBody":                      {Text: "HTTP solo sirve la shell y los archivos estaticos. Las acciones de cuenta ahora pasan por el tunel del servicio de chat."},
			"auth.loginTitle":                    {Text: "Iniciar sesion"},
			"auth.loginBody":                     {Text: "Usa tu cuenta local del experimento para continuar el espacio actual."},
			"auth.signupTitle":                   {Text: "Crear cuenta"},
			"auth.signupBody":                    {Text: "Crea una cuenta local para este experimento. No hay paginas REST separadas para autenticacion."},
			"auth.signIn":                        {Text: "Iniciar sesion"},
			"auth.createAccount":                 {Text: "Crear cuenta"},
			"auth.switchToSignup":                {Text: "Necesitas una cuenta? Creala"},
			"auth.switchToLogin":                 {Text: "Ya tienes una cuenta? Inicia sesion"},
			"auth.displayName":                   {Text: "Nombre visible"},
			"auth.displayNamePlaceholder":        {Text: "Nombre visible opcional"},
			"auth.email":                         {Text: "Correo"},
			"auth.emailPlaceholder":              {Text: "tu@ejemplo.com"},
			"auth.password":                      {Text: "Contrasena"},
			"auth.passwordPlaceholder":           {Text: "Escribe una contrasena"},
			"auth.passwordHelp":                  {Text: "Usa al menos 8 caracteres para registrarte."},
			"auth.logout":                        {Text: "Cerrar sesion"},
			"empty.heading":                      {Text: "Explora el experimento de UI Go WASM"},
			"empty.body":                         {Text: "Pregunta sobre Go, Golang, WebAssembly, patrones de UI o tu proximo experimento de framework."},
			"input.placeholder":                  {Text: "Escribe al experimento de GoWebComponents..."},
			"input.disclaimer":                   {Text: "La salida experimental de Go/WASM UI puede ser incorrecta. Verifica los detalles importantes."},
			"input.threadTotal":                  {Text: "Total del hilo {cost}"},
			"input.threadTotalPartial":           {Text: "Total del hilo {cost} (parcial)"},
			"controls.provider":                  {Text: "Proveedor"},
			"controls.model":                     {Text: "Modelo"},
			"controls.intelligence":              {Text: "Inteligencia"},
			"sidebar.newChat":                    {Text: "Nuevo chat"},
			"sidebar.noConversations":            {Text: "Todavia no hay conversaciones"},
			"sidebar.emptyConversation":          {Text: "Conversacion vacia"},
			"sidebar.editSettings":               {Text: "Editar ajustes"},
			"sidebar.deleteConversation":         {Text: "Eliminar conversacion"},
			"quote.prompt":                       {Text: "Citar texto"},
			"canvas.badge":                       {Text: "Canvas"},
			"canvas.title":                       {Text: "Vista previa"},
			"canvas.caption":                     {Text: "Ultimo bloque ```canvas"},
			"message.thinking":                   {Text: "Pensando"},
			"message.live":                       {Text: "En vivo"},
			"message.switchedTo":                 {Text: "cambio a {model}"},
			"message.cancel":                     {Text: "Cancelar"},
			"message.saveResend":                 {Text: "Guardar y reenviar"},
			"message.copy":                       {Text: "Copiar"},
			"message.edit":                       {Text: "Editar"},
			"message.fork":                       {Text: "Ramificar"},
			"assistant.loading":                  {Text: "Cargando"},
			"assistant.pause":                    {Text: "Pausar"},
			"assistant.play":                     {Text: "Reproducir"},
			"assistant.stop":                     {Text: "Detener"},
			"assistant.speechUnavailable":        {Text: "Voz no disponible"},
			"modal.deleteTitle":                  {Text: "Eliminar conversacion?"},
			"modal.deleteBody":                   {Text: "Esto eliminara permanentemente este chat y todos sus mensajes. No se puede deshacer."},
			"modal.settingsTitle":                {Text: "Ajustes"},
			"modal.displayName":                  {Text: "Nombre visible"},
			"modal.displayNamePlaceholder":       {Text: "Escribe tu nombre..."},
			"modal.aiTone":                       {Text: "Tono de IA"},
			"modal.intelligence":                 {Text: "Razonamiento"},
			"modal.intelligenceHelp":             {Text: "Elige cuanto razonamiento debe usar el asistente antes de responder."},
			"modal.intelligenceUnavailable":      {Text: "El modelo seleccionado actualmente no expone controles de razonamiento."},
			"modal.systemPrompt":                 {Text: "Prompt del sistema"},
			"modal.systemPromptPlaceholder":      {Text: "Agrega instrucciones duraderas sobre como debe responder el asistente..."},
			"modal.systemPromptHelp":             {Text: "Estas instrucciones se agregan al prompt base del asistente para tu cuenta."},
			"modal.memories":                     {Text: "Preferencias recordadas"},
			"modal.memoriesHelp":                 {Text: "Revisa y edita los detalles del perfil que el asistente recuerda sobre ti."},
			"modal.memoriesEmpty":                {Text: "Todavia no hay preferencias recordadas."},
			"modal.memoryAdd":                    {Text: "Agregar memoria"},
			"modal.memoryItem":                   {Text: "Elemento del perfil"},
			"modal.memoryDelete":                 {Text: "Eliminar"},
			"modal.memoryCategoryPlaceholder":    {Text: "Categoria, por ejemplo preferencia o proyecto"},
			"modal.memorySummaryPlaceholder":     {Text: "Resumen corto"},
			"modal.memoryDetailPlaceholder":      {Text: "Detalle opcional"},
			"modal.memoryReasonPlaceholder":      {Text: "Por que importa"},
			"modal.language":                     {Text: "Idioma"},
			"modal.save":                         {Text: "Guardar"},
			"tone.balanced.label":                {Text: "Equilibrado"},
			"tone.balanced.desc":                 {Text: "Util y conversacional"},
			"tone.professional.label":            {Text: "Profesional"},
			"tone.professional.desc":             {Text: "Formal y preciso"},
			"tone.concise.label":                 {Text: "Conciso"},
			"tone.concise.desc":                  {Text: "Breve y directo"},
			"thinking.off":                       {Text: "Off"},
			"thinking.low":                       {Text: "Bajo"},
			"thinking.medium":                    {Text: "Med"},
			"thinking.high":                      {Text: "Alto"},
			"error.genericPrefix":                {Text: "ERROR"},
			"error.receivePrefix":                {Text: "ERROR recv:"},
			"tts.audioPlaybackFailed":            {Text: "La reproduccion de audio fallo"},
			"tts.audioUnavailable":               {Text: "La salida de audio no esta disponible en este navegador"},
			"tts.audioCouldNotStart":             {Text: "La reproduccion de audio no pudo iniciarse"},
			"tts.audioCouldNotResume":            {Text: "La reproduccion de audio no pudo reanudarse"},
			"tts.noText":                         {Text: "No hay nada para leer en voz alta"},
			"tts.connectionNotReady":             {Text: "La conexion del chat todavia no esta lista"},
			"tts.speechSynthesisFailed":          {Text: "La sintesis de voz fallo"},
			"tts.speechSynthesisReturnedNoAudio": {Text: "La sintesis de voz no devolvio audio"},
		},
	})
	bundle.Register("fr", i18n.Catalog{
		chatI18nNamespace: {
			"auth.badge":                         {Text: "Auth GWC"},
			"auth.connecting":                    {Text: "Connexion au pont gRPC local..."},
			"auth.resolving":                     {Text: "Resolution de la session enregistree..."},
			"auth.heroTitle":                     {Text: "Utilisez l'experience de chat sans site de connexion separe."},
			"auth.heroBody":                      {Text: "La route racine reste maintenant dans l'application Go/WASM. Connectez-vous ou creez un compte via gRPC et gardez tout dans une seule interface."},
			"auth.grpcBadge":                     {Text: "gRPC uniquement"},
			"auth.grpcBody":                      {Text: "HTTP ne sert plus que la shell et les assets statiques. Les actions de compte passent maintenant par le tunnel du service de chat."},
			"auth.loginTitle":                    {Text: "Connexion"},
			"auth.loginBody":                     {Text: "Utilisez votre compte local d'experience pour reprendre l'espace courant."},
			"auth.signupTitle":                   {Text: "Creer un compte"},
			"auth.signupBody":                    {Text: "Creez un compte local pour cette experience. Il n'y a plus de pages REST de connexion separees."},
			"auth.signIn":                        {Text: "Connexion"},
			"auth.createAccount":                 {Text: "Creer un compte"},
			"auth.switchToSignup":                {Text: "Besoin d'un compte ? Creez-en un"},
			"auth.switchToLogin":                 {Text: "Vous avez deja un compte ? Connectez-vous"},
			"auth.displayName":                   {Text: "Nom affiche"},
			"auth.displayNamePlaceholder":        {Text: "Nom affiche facultatif"},
			"auth.email":                         {Text: "Email"},
			"auth.emailPlaceholder":              {Text: "vous@exemple.com"},
			"auth.password":                      {Text: "Mot de passe"},
			"auth.passwordPlaceholder":           {Text: "Entrez un mot de passe"},
			"auth.passwordHelp":                  {Text: "Utilisez au moins 8 caracteres pour l'inscription."},
			"auth.logout":                        {Text: "Se deconnecter"},
			"empty.heading":                      {Text: "Explorez l'experience UI Go WASM"},
			"empty.body":                         {Text: "Posez des questions sur Go, Golang, WebAssembly, les patterns UI ou votre prochaine experience de framework."},
			"input.placeholder":                  {Text: "Envoyez un message a l'experience GoWebComponents..."},
			"input.disclaimer":                   {Text: "La sortie experimentale Go/WASM UI peut etre incorrecte. Verifiez les details importants."},
			"input.threadTotal":                  {Text: "Total du fil {cost}"},
			"input.threadTotalPartial":           {Text: "Total du fil {cost} (partiel)"},
			"controls.provider":                  {Text: "Fournisseur"},
			"controls.model":                     {Text: "Modele"},
			"controls.intelligence":              {Text: "Intelligence"},
			"sidebar.newChat":                    {Text: "Nouveau chat"},
			"sidebar.noConversations":            {Text: "Aucune conversation pour le moment"},
			"sidebar.emptyConversation":          {Text: "Conversation vide"},
			"sidebar.editSettings":               {Text: "Modifier les reglages"},
			"sidebar.deleteConversation":         {Text: "Supprimer la conversation"},
			"quote.prompt":                       {Text: "Citer le texte"},
			"canvas.badge":                       {Text: "Canvas"},
			"canvas.title":                       {Text: "Apercu en direct"},
			"canvas.caption":                     {Text: "Dernier bloc ```canvas"},
			"message.thinking":                   {Text: "Reflexion"},
			"message.live":                       {Text: "En direct"},
			"message.switchedTo":                 {Text: "passe a {model}"},
			"message.cancel":                     {Text: "Annuler"},
			"message.saveResend":                 {Text: "Enregistrer et renvoyer"},
			"message.copy":                       {Text: "Copier"},
			"message.edit":                       {Text: "Modifier"},
			"message.fork":                       {Text: "Fork"},
			"assistant.loading":                  {Text: "Chargement"},
			"assistant.pause":                    {Text: "Pause"},
			"assistant.play":                     {Text: "Lire"},
			"assistant.stop":                     {Text: "Arreter"},
			"assistant.speechUnavailable":        {Text: "Voix indisponible"},
			"modal.deleteTitle":                  {Text: "Supprimer la conversation ?"},
			"modal.deleteBody":                   {Text: "Cela supprimera definitivement ce chat et tous ses messages. Cette action est irreversible."},
			"modal.settingsTitle":                {Text: "Reglages"},
			"modal.displayName":                  {Text: "Nom affiche"},
			"modal.displayNamePlaceholder":       {Text: "Entrez votre nom..."},
			"modal.aiTone":                       {Text: "Ton IA"},
			"modal.intelligence":                 {Text: "Raisonnement"},
			"modal.intelligenceHelp":             {Text: "Choisissez le niveau de raisonnement que l'assistant doit utiliser avant de repondre."},
			"modal.intelligenceUnavailable":      {Text: "Le modele selectionne n'expose pas de controles de raisonnement."},
			"modal.systemPrompt":                 {Text: "Prompt systeme"},
			"modal.systemPromptPlaceholder":      {Text: "Ajoutez des instructions durables sur la maniere dont l'assistant doit repondre..."},
			"modal.systemPromptHelp":             {Text: "Ces instructions sont ajoutees au prompt de base de l'assistant pour votre compte."},
			"modal.memories":                     {Text: "Preferences memorisees"},
			"modal.memoriesHelp":                 {Text: "Examinez et modifiez les details du profil dont l'assistant se souvient a votre sujet."},
			"modal.memoriesEmpty":                {Text: "Aucune preference memorisee pour le moment."},
			"modal.memoryAdd":                    {Text: "Ajouter"},
			"modal.memoryItem":                   {Text: "Element du profil"},
			"modal.memoryDelete":                 {Text: "Supprimer"},
			"modal.memoryCategoryPlaceholder":    {Text: "Categorie, par exemple preference ou projet"},
			"modal.memorySummaryPlaceholder":     {Text: "Resume court"},
			"modal.memoryDetailPlaceholder":      {Text: "Detail facultatif"},
			"modal.memoryReasonPlaceholder":      {Text: "Pourquoi c'est utile"},
			"modal.language":                     {Text: "Langue"},
			"modal.save":                         {Text: "Enregistrer"},
			"tone.balanced.label":                {Text: "Equilibre"},
			"tone.balanced.desc":                 {Text: "Utile et conversationnel"},
			"tone.professional.label":            {Text: "Professionnel"},
			"tone.professional.desc":             {Text: "Formel et precis"},
			"tone.concise.label":                 {Text: "Concis"},
			"tone.concise.desc":                  {Text: "Bref et direct"},
			"thinking.off":                       {Text: "Off"},
			"thinking.low":                       {Text: "Bas"},
			"thinking.medium":                    {Text: "Moy"},
			"thinking.high":                      {Text: "Haut"},
			"error.genericPrefix":                {Text: "ERREUR"},
			"error.receivePrefix":                {Text: "ERREUR recv:"},
			"tts.audioPlaybackFailed":            {Text: "La lecture audio a echoue"},
			"tts.audioUnavailable":               {Text: "La sortie audio n'est pas disponible dans ce navigateur"},
			"tts.audioCouldNotStart":             {Text: "La lecture audio n'a pas pu demarrer"},
			"tts.audioCouldNotResume":            {Text: "La lecture audio n'a pas pu reprendre"},
			"tts.noText":                         {Text: "Aucun texte a lire a voix haute"},
			"tts.connectionNotReady":             {Text: "La connexion du chat n'est pas encore prete"},
			"tts.speechSynthesisFailed":          {Text: "La synthese vocale a echoue"},
			"tts.speechSynthesisReturnedNoAudio": {Text: "La synthese vocale n'a retourne aucun audio"},
		},
	})
	return bundle
}()

func chatWizardRoot() ui.Node {
	locale := i18n.UseLocale(i18n.LocaleOptions{
		InitialLocale:    "en",
		SupportedLocales: supportedChatLocaleIDs(),
		FallbackLocale:   "en",
		PersistenceKey:   chatLocalePersistenceKey,
		DetectBrowser:    true,
	})
	return i18n.Provider(i18n.ProviderProps{
		Locale: locale,
		Bundle: chatWizardBundle,
		Child:  ui.CreateElement(App),
	})
}

func supportedChatLocaleIDs() []string {
	locales := make([]string, 0, len(availableLocales))
	for _, option := range availableLocales {
		locales = append(locales, option.ID)
	}
	return locales
}

func normalizeChatLocaleID(locale string) string {
	normalized := i18n.NormalizeLocale(locale)
	for _, option := range availableLocales {
		if option.ID == normalized {
			return option.ID
		}
	}
	return availableLocales[0].ID
}

func localeLabel(id string) string {
	for _, option := range availableLocales {
		if option.ID == id {
			return option.NativeLabel
		}
	}
	return strings.TrimSpace(id)
}

func toneLabel(intl i18n.Runtime, id string) string {
	return intl.T(chatI18nNamespace, "tone."+id+".label")
}

func toneDescription(intl i18n.Runtime, id string) string {
	return intl.T(chatI18nNamespace, "tone."+id+".desc")
}

func thinkingEffortLabel(intl i18n.Runtime, id string) string {
	return intl.T(chatI18nNamespace, "thinking."+id)
}

func thoughtHeadingLabel(intl i18n.Runtime, heading string) string {
	if strings.EqualFold(strings.TrimSpace(heading), "thinking") {
		return intl.T(chatI18nNamespace, "message.thinking")
	}
	return heading
}
