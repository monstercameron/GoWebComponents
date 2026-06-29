-- Bill Splitter — Elm (The Elm Architecture).
--
-- Note: Elm centralizes ALL state in one Model by design — there is no separate
-- "shared state" primitive; theme/roundUp are just fields. That uniformity is the
-- comparison point for Elm. Build: elm make src/Main.elm --output=main.js
-- Status: to-spec scaffold, not build-verified here.


module Main exposing (main)

import Browser
import Html exposing (Html, button, div, footer, h1, h2, header, input, label, li, main_, p, section, span, text, ul)
import Html.Attributes exposing (attribute, class, classList, disabled, placeholder, step, type_, value)
import Html.Events exposing (onClick, onInput)


type alias Model =
    { bill : Float, tip : Float, people : Int, theme : String, roundUp : Bool }


init : Model
init =
    { bill = 0, tip = 18, people = 1, theme = "light", roundUp = False }


type Msg
    = SetBill String
    | SetTip String
    | Preset Float
    | Inc
    | Dec
    | ToggleTheme
    | ToggleRound


update : Msg -> Model -> Model
update msg m =
    case msg of
        SetBill s ->
            { m | bill = parseNum s }

        SetTip s ->
            { m | tip = parseNum s }

        Preset p ->
            { m | tip = p }

        Inc ->
            { m | people = m.people + 1 }

        Dec ->
            { m | people = max 1 (m.people - 1) }

        ToggleTheme ->
            { m | theme = ifElse (m.theme == "dark") "light" "dark" }

        ToggleRound ->
            { m | roundUp = not m.roundUp }



-- derived


tipAmount : Model -> Float
tipAmount m =
    m.bill * m.tip / 100


total : Model -> Float
total m =
    m.bill + tipAmount m


perPersonRaw : Model -> Float
perPersonRaw m =
    if m.people > 0 then
        total m / toFloat m.people

    else
        0


perPerson : Model -> Float
perPerson m =
    if m.roundUp then
        toFloat (ceiling (perPersonRaw m))

    else
        perPersonRaw m


totalCollected : Model -> Float
totalCollected m =
    if m.roundUp then
        perPerson m * toFloat m.people

    else
        total m


roundingExtra : Model -> Float
roundingExtra m =
    max 0 (totalCollected m - total m)


effTip : Model -> Float
effTip m =
    if m.bill > 0 then
        (totalCollected m - m.bill) / m.bill * 100

    else
        m.tip



-- view


presets : List Float
presets =
    [ 10, 15, 18, 20, 25 ]


view : Model -> Html Msg
view m =
    main_ [ class "bs-app", attribute "data-theme" m.theme ]
        [ header [ class "bs-header" ]
            [ h1 [ class "bs-title" ] [ text "Bill Splitter" ]
            , div [ class "bs-header-actions" ]
                [ button [ class "bs-toggle", attribute "aria-pressed" (boolStr m.roundUp), onClick ToggleRound ] [ text "Round up" ]
                , button [ class "bs-toggle", attribute "aria-pressed" (boolStr (m.theme == "dark")), onClick ToggleTheme ] [ text "Dark" ]
                ]
            ]
        , section [ class "bs-card bs-inputs" ]
            [ label [ class "bs-field" ]
                [ span [ class "bs-label" ] [ text "Bill amount" ]
                , div [ class "bs-input-wrap" ]
                    [ span [ class "bs-prefix" ] [ text "$" ]
                    , input [ class "bs-input", type_ "number", step "0.01", value (numStr m.bill), onInput SetBill ] []
                    ]
                ]
            , div [ class "bs-field" ]
                [ span [ class "bs-label" ] [ text "Tip" ]
                , div [ class "bs-presets" ]
                    (List.map (presetButton m) presets
                        ++ [ input [ class "bs-preset-custom", type_ "number", placeholder "Custom %", value (numStr m.tip), onInput SetTip ] [] ]
                    )
                ]
            , div [ class "bs-field" ]
                [ span [ class "bs-label" ] [ text "People" ]
                , div [ class "bs-stepper" ]
                    [ button [ class "bs-step", disabled (m.people <= 1), onClick Dec ] [ text "−" ]
                    , span [ class "bs-count" ] [ text (String.fromInt m.people) ]
                    , button [ class "bs-step", onClick Inc ] [ text "+" ]
                    ]
                ]
            ]
        , section [ class "bs-card bs-results" ]
            [ resultRow "Tip" (usd (tipAmount m))
            , resultRow "Total" (usd (total m))
            , div [ class "bs-result-hero" ]
                [ span [ class "bs-result-hero-label" ] [ text "Per person" ]
                , span [ class "bs-result-hero-value" ] [ text (usd (perPerson m)) ]
                ]
            , if m.roundUp && roundingExtra m > 0 then
                p [ class "bs-note" ] [ text ("Rounding up collects " ++ usd (roundingExtra m) ++ " extra · effective tip " ++ oneDecimal (effTip m) ++ "%") ]

              else
                text ""
            , if m.bill <= 0 then
                p [ class "bs-empty" ] [ text "Enter a bill amount to begin." ]

              else
                text ""
            ]
        , section [ class "bs-card bs-breakdown" ]
            [ h2 [ class "bs-subtitle" ] [ text "Per-person breakdown" ]
            , ul [ class "bs-people" ]
                (List.map (\n -> li [ class "bs-person" ] [ span [] [ text ("Person " ++ String.fromInt n) ], span [] [ text (usd (perPerson m)) ] ])
                    (List.range 1 m.people)
                )
            ]
        , footer [ class "bs-footer" ]
            [ text ("Splitting " ++ usd (total m) ++ " between " ++ String.fromInt m.people ++ " · " ++ m.theme ++ " theme") ]
        ]


presetButton : Model -> Float -> Html Msg
presetButton m p =
    button
        [ classList [ ( "bs-preset", True ), ( "bs-preset--active", m.tip == p ) ]
        , onClick (Preset p)
        ]
        [ text (String.fromFloat p ++ "%") ]


resultRow : String -> String -> Html Msg
resultRow lbl val =
    div [ class "bs-result-row" ] [ span [] [ text lbl ], span [] [ text val ] ]



-- helpers


main : Program () Model Msg
main =
    Browser.sandbox { init = init, update = update, view = view }


ifElse : Bool -> a -> a -> a
ifElse c a b =
    if c then
        a

    else
        b


boolStr : Bool -> String
boolStr b =
    ifElse b "true" "false"


parseNum : String -> Float
parseNum s =
    case String.toFloat s of
        Just v ->
            if v >= 0 then
                v

            else
                0

        Nothing ->
            0


numStr : Float -> String
numStr v =
    if v == 0 then
        ""

    else
        String.fromFloat v


oneDecimal : Float -> String
oneDecimal v =
    String.fromFloat (toFloat (round (v * 10)) / 10)


usd : Float -> String
usd v =
    let
        cents =
            max 0 (round (v * 100))

        dollars =
            cents // 100

        frac =
            modBy 100 cents

        grouped =
            groupThousands (String.fromInt dollars)

        fracStr =
            if frac < 10 then
                "0" ++ String.fromInt frac

            else
                String.fromInt frac
    in
    "$" ++ grouped ++ "." ++ fracStr


groupThousands : String -> String
groupThousands s =
    let
        chars =
            String.toList s

        n =
            List.length chars

        withCommas =
            List.indexedMap
                (\i c ->
                    if i > 0 && modBy 3 (n - i) == 0 then
                        "," ++ String.fromChar c

                    else
                        String.fromChar c
                )
                chars
    in
    String.concat withCommas
