module Sharecrop.Ui exposing (..)

import Html exposing (Attribute, Html, button, h1, h2, h3, input, label, p, pre, span, text, textarea)
import Html.Attributes exposing (attribute, class, type_)
import Html.Events exposing (onClick)


testId : String -> Attribute msg
testId value =
    attribute "data-testid" value


card : List (Html msg) -> Html msg
card children =
    Html.div [ class "space-y-4 rounded-lg border border-slate-200 bg-white p-4 shadow-sm sm:p-6" ] children


sectionTitle : String -> Html msg
sectionTitle title =
    h2 [ class "text-lg font-medium" ] [ text title ]


{-| A collapsible section built on the native `<details>`/`<summary>`
elements, so hiding/revealing a section needs no Elm model or message wiring.
Use for secondary sections on pages that stack several full panels (filters,
admin queues) so the page reads short by default and expands on demand.
-}
disclosure : String -> Bool -> String -> List (Html msg) -> Html msg
disclosure identifier openByDefault title children =
    Html.node "details"
        (if openByDefault then
            [ attribute "open" "" ]

         else
            []
        )
        [ Html.node "summary" [ class "cursor-pointer select-none text-lg font-medium marker:text-slate-400", testId identifier ] [ text title ]
        , Html.div [ class "mt-3 space-y-4" ] children
        ]


pageTitle : String -> Html msg
pageTitle title =
    h1 [ class "text-3xl font-semibold" ] [ text title ]


{-| A dropdown menu for the nav bar's grouped links: the panel is absolutely
positioned below the trigger so it floats over page content rather than
pushing it down. `alignRight` hangs the panel from the trigger's right edge
instead of its left, for menus near the right side of the screen that would
otherwise overflow off-screen. `isActive` highlights the trigger itself (like
the active page's nav link) when the current page is one of this menu's
items, so closing the menu doesn't lose the "you are here" signal.

Deliberately Elm-controlled (`isOpen`/`onToggle`) rather than a native
`<details>`/`<summary>`: a menu item is a link that navigates to a new page,
and a native details element has no way for Elm to close it again once
that navigation completes — left open, its floating panel would sit over
the new page and intercept clicks on whatever is underneath it. The caller
is expected to reset the open menu to `Nothing` on every route change.
-}
navMenu : String -> Bool -> Bool -> Bool -> msg -> String -> List (Html msg) -> Html msg
navMenu identifier alignRight isActive isOpen onToggle triggerText children =
    Html.div [ class "relative inline-block" ]
        (button
            [ type_ "button"
            , onClick onToggle
            , attribute "aria-expanded"
                (if isOpen then
                    "true"

                 else
                    "false"
                )
            , attribute "aria-haspopup" "true"
            , class
                ((if isActive then
                    primaryButtonClass

                  else
                    secondaryButtonClass
                 )
                    ++ " select-none"
                )
            , testId identifier
            ]
            [ text (triggerText ++ " \u{25BE}") ]
            :: (if isOpen then
                    [ Html.div
                        [ class
                            ("absolute top-full z-20 mt-2 flex min-w-[190px] flex-col gap-2 rounded-lg border border-slate-200 bg-white p-2 shadow-lg "
                                ++ (if alignRight then
                                        "right-0"

                                    else
                                        "left-0"
                                   )
                            )
                        ]
                        children
                    ]

                else
                    []
               )
        )


{-| A heading that also reports a live count, e.g. "Teams (3)".
-}
sectionTitleWithCount : String -> Int -> String -> Html msg
sectionTitleWithCount title count identifier =
    h3 [ class "text-lg font-medium", testId identifier ]
        [ text (title ++ " (" ++ String.fromInt count ++ ")") ]


{-| A two-state toggle button for a mutually-exclusive chooser group (reward
kind, participation policy, visibility, assignee scope, etc). Reports its
pressed state via `aria-pressed` since these are toggle buttons, not links or
form submits.
-}
chooserButton : Bool -> msg -> String -> String -> Html msg
chooserButton isSelected msg identifier labelText =
    if isSelected then
        primaryButton [ type_ "button", onClick msg, attribute "aria-pressed" "true", testId identifier ] labelText

    else
        secondaryButton [ type_ "button", onClick msg, attribute "aria-pressed" "false", testId identifier ] labelText


label_ : String -> Html msg
label_ value =
    p [ class "text-sm uppercase tracking-wide text-slate-600" ] [ text value ]


fieldLabel : String -> List (Html msg) -> Html msg
fieldLabel labelText controls =
    label [ class "block min-w-0 grow space-y-1 text-sm font-medium text-slate-700" ]
        (span [] [ text labelText ] :: controls)


checkbox : List (Attribute msg) -> String -> Html msg
checkbox attrs labelText =
    label [ class "flex items-center gap-2 text-sm text-slate-700" ]
        [ input (class checkboxClass :: type_ "checkbox" :: attrs) []
        , span [] [ text labelText ]
        ]


primaryButton : List (Attribute msg) -> String -> Html msg
primaryButton attrs labelText =
    button (class primaryButtonClass :: attrs) [ text labelText ]


secondaryButton : List (Attribute msg) -> String -> Html msg
secondaryButton attrs labelText =
    button (class secondaryButtonClass :: attrs) [ text labelText ]


dangerButton : List (Attribute msg) -> String -> Html msg
dangerButton attrs labelText =
    button (class dangerButtonClass :: attrs) [ text labelText ]


textInput : List (Attribute msg) -> Html msg
textInput attrs =
    input (class fieldClass :: attrs) []


{-| Like textInput, but switches to the invalid-field style when the first
argument is True - a required field the user tried to submit empty. Color
alone never carries this signal (WCAG 1.4.1): pair it with fieldError below
the input, not with this alone.
-}
textInputToned : Bool -> List (Attribute msg) -> Html msg
textInputToned invalid attrs =
    input (class (fieldToneClass invalid) :: attrs) []


textarea_ : List (Attribute msg) -> Html msg
textarea_ attrs =
    textarea (class textareaClass :: attrs) []


textareaToned : Bool -> List (Attribute msg) -> Html msg
textareaToned invalid attrs =
    textarea (class (textareaToneClass invalid) :: attrs) []


fieldToneClass : Bool -> String
fieldToneClass invalid =
    if invalid then
        "w-full min-h-[44px] rounded-md border border-red-400 bg-red-50 px-3 py-2 text-sm focus:border-red-500 focus:outline-none focus-visible:ring-2 focus-visible:ring-red-400 focus-visible:ring-offset-1"

    else
        fieldClass


textareaToneClass : Bool -> String
textareaToneClass invalid =
    if invalid then
        "w-full rounded-md border border-red-400 bg-red-50 px-3 py-2 font-mono text-sm focus:border-red-500 focus:outline-none focus-visible:ring-2 focus-visible:ring-red-400 focus-visible:ring-offset-1"

    else
        textareaClass


{-| An inline required-field error: paired with textInputToned/textareaToned
so the field is never identified by color alone, per WCAG 1.4.1.
-}
fieldError : String -> Html msg
fieldError message =
    p [ class "flex items-center gap-1.5 text-xs text-red-700" ]
        [ span [ class "inline-block h-3.5 w-3.5 shrink-0 rounded-full border border-red-700 text-center text-[9px] font-bold leading-[13px]" ] [ text "!" ]
        , text message
        ]


badge : String -> Html msg
badge value =
    badgeVariant "neutral" value


{-| A status pill in one of a small set of semantic tones, so state is
conveyed by more than color alone (the text itself still names the state).
Each tone's Tailwind pair is chosen to keep the pill's text at or above a
4.5:1 contrast ratio against its own background (WCAG AA for normal text):

  - neutral: `slate-700` (#334155) on `slate-100` (#f1f5f9) — ~8.4:1
  - success: `green-800` (#166534) on `green-100` (#dcfce7) — ~7.4:1
  - warning: `amber-900` (#78350f) on `amber-100` (#fef3c7) — ~8.9:1
  - danger: `red-800` (#991b1b) on `red-100` (#fee2e2) — ~6.8:1

-}
badgeVariant : String -> String -> Html msg
badgeVariant tone value =
    span [ class ("inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium " ++ badgeToneClass tone) ] [ text value ]


badgeToneClass : String -> String
badgeToneClass tone =
    case tone of
        "success" ->
            "bg-green-100 text-green-800"

        "info" ->
            "bg-blue-100 text-blue-800"

        "warning" ->
            "bg-amber-100 text-amber-900"

        "danger" ->
            "bg-red-100 text-red-800"

        _ ->
            "bg-slate-100 text-slate-700"


codeBlock : List (Attribute msg) -> String -> Html msg
codeBlock attrs value =
    pre (class codeBlockClass :: attrs) [ text value ]


errorText : String -> String -> Html msg
errorText identifier message =
    p [ class "text-sm text-red-600", testId identifier ] [ text message ]


noteText : String -> String -> Html msg
noteText identifier message =
    p [ class "text-sm text-slate-600", testId identifier ] [ text message ]



-- Shared class strings


primaryButtonClass : String
primaryButtonClass =
    "inline-flex min-h-[44px] items-center justify-center rounded-md bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-700 disabled:opacity-50"


secondaryButtonClass : String
secondaryButtonClass =
    "inline-flex min-h-[44px] items-center justify-center rounded-md border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100"


dangerButtonClass : String
dangerButtonClass =
    "inline-flex min-h-[44px] items-center justify-center rounded-md border border-red-300 px-4 py-2 text-sm font-medium text-red-700 hover:bg-red-50"


fieldClass : String
fieldClass =
    "w-full min-h-[44px] rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-slate-500 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-500 focus-visible:ring-offset-1"


checkboxClass : String
checkboxClass =
    "h-4 w-4 rounded border-slate-400 text-slate-900 focus:ring-2 focus:ring-slate-500"


textareaClass : String
textareaClass =
    "w-full rounded-md border border-slate-300 px-3 py-2 font-mono text-sm focus:border-slate-500 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-500 focus-visible:ring-offset-1"


codeBlockClass : String
codeBlockClass =
    "whitespace-pre-wrap break-words rounded-md bg-slate-900 p-3 text-xs text-slate-100"
