module Sharecrop.Ui exposing (..)

import Html exposing (Attribute, Html, a, button, h1, h2, h3, input, label, p, pre, span, text, textarea)
import Html.Attributes exposing (attribute, class, href, type_)
import Html.Events exposing (onClick)
import Sharecrop.Sprites as Sprites


testId : String -> Attribute msg
testId value =
    attribute "data-testid" value


card : List (Html msg) -> Html msg
card children =
    Html.div [ class cardClass ] children


{-| A bordered sub-panel that sits on a card: comment entries, saved-view
boxes, dashboards. Bright field paper with a hard ink border so it reads as
a block resting on the card. Callers pass their own spacing/testid attrs.
-}
subCard : List (Attribute msg) -> List (Html msg) -> Html msg
subCard attrs children =
    Html.div (class subCardClass :: attrs) children


{-| A recessed panel for secondary content (filters, token reveals, option
groups): darker parchment with a soft line, visually "cut into" the card
rather than resting on it. Callers pass their own spacing/testid attrs.
-}
insetPanel : List (Attribute msg) -> List (Html msg) -> Html msg
insetPanel attrs children =
    Html.div (class (insetPanelClass ++ " p-4") :: attrs) children


{-| An empty-list placeholder: a small pixel sprite (aria-hidden, decorative)
above the muted message. The testid and message text stay on the paragraph
itself, exactly as the plain empty-state paragraphs had them, so tests keyed
on either keep working.
-}
emptyState : String -> String -> String -> Html msg
emptyState identifier slug message =
    Html.div [ class "flex flex-col items-center gap-3 py-6 text-center" ]
        [ Sprites.pixel slug 4
        , p [ class "text-sm text-farm-muted", testId identifier ] [ text message ]
        ]


sectionTitle : String -> Html msg
sectionTitle title =
    h2 [ class "font-display text-[0.85rem] leading-relaxed text-farm-accent-strong" ] [ text title ]


{-| A collapsible section built on the native `<details>`/`<summary>`
elements, so hiding/revealing a section needs no Elm model or message wiring.
Use for secondary sections on pages that stack several full panels (filters,
admin queues) so the page reads short by default and expands on demand.
-}
-- disclosure is a native collapsible section. openByDefault must be a value
-- that stays stable for the lifetime of the page (a literal, or a snapshot
-- taken when the page is entered) - never derive it from live form state.
-- Elm re-applies the `open` attribute whenever the value changes between
-- renders, which force-opens or force-closes the panel under the user
-- mid-edit (e.g. clearing a search field used to snap the whole Filters
-- panel shut).


disclosure : String -> Bool -> String -> List (Html msg) -> Html msg
disclosure identifier openByDefault title children =
    Html.node "details"
        (if openByDefault then
            [ attribute "open" "" ]

         else
            []
        )
        [ Html.node "summary" [ class "cursor-pointer select-none font-display text-[0.85rem] leading-relaxed text-farm-accent-strong marker:text-farm-accent", testId identifier ] [ text title ]
        , Html.div [ class "mt-3 space-y-4" ] children
        ]


{-| A compact "ⓘ" info toggle that sits beside an action and explains, on
demand, what that action does. It is a native `<details>` so it needs no app
state (the same reason `disclosure` uses one), the marker is dropped so the
summary reads as a small info chip rather than a section header, and the tone
matches the info badge. The glyph is `aria-hidden` and purely decorative - the
"What this does" label carries the meaning for assistive tech (WCAG 1.4.1).
-}
explainToggle : String -> String -> Html msg
explainToggle identifier explanation =
    Html.node "details"
        [ class "text-xs", testId identifier ]
        [ Html.node "summary"
            [ class "inline-flex w-fit cursor-pointer select-none items-center gap-1 border-2 border-farm-info bg-farm-info-soft px-2 py-0.5 font-medium text-farm-info list-none [&::-webkit-details-marker]:hidden" ]
            [ span [ attribute "aria-hidden" "true" ] [ text "ⓘ" ]
            , text "What this does"
            ]
        , p [ class "mt-1 max-w-prose text-farm-muted" ] [ text explanation ]
        ]


pageTitle : String -> Html msg
pageTitle title =
    h1 [ class "font-display text-xl leading-relaxed text-farm-accent-strong [text-shadow:2px_2px_0_var(--color-farm-surface-raised)]" ] [ text title ]


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

`identityMarker`, when present, publishes the signed-in username on the
trigger so post-deployment qualification can find the identity and open the
menu to reach the real sign-out control inside it.

`triggerExtras` renders after the trigger text inside the button - used for
the unread-count pill on the Account menu, which must stay visible while the
menu is closed.
-}
navMenu : String -> Bool -> Bool -> Bool -> msg -> String -> List (Html msg) -> Maybe String -> List (Html msg) -> Html msg
navMenu identifier alignRight isActive isOpen onToggle triggerText triggerExtras identityMarker children =
    Html.div [ class "relative inline-block" ]
        (button
            ([ type_ "button"
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
                ++ (case identityMarker of
                        Just username ->
                            [ attribute "data-shauth-user" username ]

                        Nothing ->
                            []
                   )
            )
            (text (triggerText ++ " \u{25BE}") :: triggerExtras)
            :: (if isOpen then
                    [ Html.div
                        [ class
                            ("absolute top-full z-20 mt-2 flex min-w-[190px] flex-col gap-2 border-[3px] border-farm-line bg-farm-surface-raised p-2 shadow-pixel "
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
    h3 [ class "font-display text-[0.85rem] leading-relaxed text-farm-accent-strong", testId identifier ]
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
    p [ class "font-display text-[0.55rem] uppercase leading-relaxed tracking-wide text-farm-accent-strong" ] [ text value ]


fieldLabel : String -> List (Html msg) -> Html msg
fieldLabel labelText controls =
    label [ class "block min-w-0 grow space-y-1 text-sm font-medium text-farm-ink" ]
        (span [] [ text labelText ] :: controls)


checkbox : List (Attribute msg) -> String -> Html msg
checkbox attrs labelText =
    label [ class "flex items-center gap-2 text-sm text-farm-ink" ]
        [ input (class checkboxClass :: type_ "checkbox" :: attrs) []
        , span [] [ text labelText ]
        ]


primaryButton : List (Attribute msg) -> String -> Html msg
primaryButton attrs labelText =
    button (class primaryButtonClass :: attrs) [ text labelText ]


secondaryButton : List (Attribute msg) -> String -> Html msg
secondaryButton attrs labelText =
    button (class secondaryButtonClass :: attrs) [ text labelText ]


secondaryLink : List (Attribute msg) -> String -> String -> Html msg
secondaryLink attrs destination labelText =
    a (class secondaryButtonClass :: href destination :: attrs) [ text labelText ]


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
        "w-full min-h-[44px] border-[3px] border-farm-danger bg-farm-danger-soft px-3 py-2 text-sm text-farm-ink"

    else
        fieldClass


textareaToneClass : Bool -> String
textareaToneClass invalid =
    if invalid then
        "w-full border-[3px] border-farm-danger bg-farm-danger-soft px-3 py-2 font-mono text-sm text-farm-ink"

    else
        textareaClass


{-| An inline required-field error: paired with textInputToned/textareaToned
so the field is never identified by color alone, per WCAG 1.4.1.
-}
fieldError : String -> Html msg
fieldError message =
    p [ class "flex items-center gap-1.5 text-xs text-farm-danger" ]
        [ span [ class "inline-block h-3.5 w-3.5 shrink-0 border border-farm-danger text-center text-[9px] font-bold leading-[13px]" ] [ text "!" ]
        , text message
        ]


badge : String -> Html msg
badge value =
    badgeVariant "neutral" value


{-| A status pill in one of a small set of semantic tones, so state is
conveyed by more than color alone (the text itself still names the state).
See badgeToneClass for the tone-to-contrast-ratio breakdown.
-}
badgeVariant : String -> String -> Html msg
badgeVariant tone value =
    span [ class ("inline-flex items-center border-2 border-farm-line px-2 py-0.5 text-xs font-semibold " ++ badgeToneClass tone) ] [ text value ]


{-| Like badgeVariant, with a small leading glyph. The glyph is
`aria-hidden` and purely decorative - the badge's own text already names
the state, so nothing is conveyed by the icon alone (WCAG 1.4.1).
-}
badgeVariantWithIcon : String -> String -> String -> Html msg
badgeVariantWithIcon tone icon value =
    span [ class ("inline-flex items-center gap-1 border-2 border-farm-line px-2 py-0.5 text-xs font-semibold " ++ badgeToneClass tone) ]
        [ span [ attribute "aria-hidden" "true" ] [ text icon ]
        , text value
        ]


{-| Tone-to-class pairs from the farm palette, each measured (WCAG relative
luminance) at or above a 4.5:1 contrast ratio against its own background
(AA for normal text):

  - neutral: `farm-ink` (#2a2118) on `farm-surface` (#f4ecd0) — 13.36:1
  - success: `farm-success` (#1e4d0d) on `farm-success-soft` (#dcebc4) — 7.88:1
  - info: `farm-info` (#1d4f80) on `farm-info-soft` (#d8e7f6) — 6.72:1
  - warning: `farm-warning` (#8a4310) on `farm-warning-soft` (#f9e0c8) — 5.72:1
  - danger: `farm-danger` (#8f2318) on `farm-danger-soft` (#f6d9d4) — 6.52:1
  - reward: `farm-reward` (#6b4a08) on `farm-reward-soft` (#f7e7b4) — 6.53:1

-}
badgeToneClass : String -> String
badgeToneClass tone =
    case tone of
        "success" ->
            "bg-farm-success-soft text-farm-success"

        "info" ->
            "bg-farm-info-soft text-farm-info"

        "warning" ->
            "bg-farm-warning-soft text-farm-warning"

        "danger" ->
            "bg-farm-danger-soft text-farm-danger"

        "reward" ->
            "bg-farm-reward-soft text-farm-reward"

        _ ->
            "bg-farm-surface text-farm-ink"


codeBlock : List (Attribute msg) -> String -> Html msg
codeBlock attrs value =
    pre (class codeBlockClass :: attrs) [ text value ]


errorText : String -> String -> Html msg
errorText identifier message =
    p [ class "text-sm text-farm-danger", testId identifier ] [ text message ]


noteText : String -> String -> Html msg
noteText identifier message =
    p [ class "text-sm text-farm-muted", testId identifier ] [ text message ]


successText : String -> String -> Html msg
successText identifier message =
    p [ class "text-sm text-farm-success", testId identifier ] [ text message ]



-- Shared class strings
--
-- Keyboard focus is handled by the global `:focus-visible` outline in
-- web/styles/input.css (the hard 3px borders here would otherwise swallow
-- Tailwind's focus:border-* indicator), so these classes carry no per-class
-- focus utilities. Buttons "press" on :active by sliding into their own
-- hard shadow, like the arcade demo skin did.


buttonPressClass : String
buttonPressClass =
    "shadow-pixel-sm active:translate-x-[3px] active:translate-y-[3px] active:shadow-none disabled:translate-x-0 disabled:translate-y-0 disabled:border-dashed disabled:shadow-none disabled:opacity-70"


primaryButtonClass : String
primaryButtonClass =
    "inline-flex min-h-[44px] items-center justify-center gap-2 border-[3px] border-farm-line bg-farm-accent px-4 py-2 text-center font-display text-[0.6rem] leading-relaxed text-farm-cream hover:bg-farm-accent-strong " ++ buttonPressClass


secondaryButtonClass : String
secondaryButtonClass =
    "inline-flex min-h-[44px] items-center justify-center gap-2 border-[3px] border-farm-line bg-farm-surface-raised px-4 py-2 text-center font-display text-[0.6rem] leading-relaxed text-farm-ink hover:bg-farm-surface " ++ buttonPressClass


dangerButtonClass : String
dangerButtonClass =
    "inline-flex min-h-[44px] items-center justify-center gap-2 border-[3px] border-farm-line bg-farm-danger-soft px-4 py-2 text-center font-display text-[0.6rem] leading-relaxed text-farm-danger hover:brightness-95 " ++ buttonPressClass


fieldClass : String
fieldClass =
    "w-full min-h-[44px] border-[3px] border-farm-line bg-farm-field px-3 py-2 text-sm text-farm-ink"


checkboxClass : String
checkboxClass =
    "h-5 w-5 accent-farm-accent-strong"


textareaClass : String
textareaClass =
    "w-full border-[3px] border-farm-line bg-farm-field px-3 py-2 font-mono text-sm text-farm-ink"


codeBlockClass : String
codeBlockClass =
    "whitespace-pre-wrap break-words border-[3px] border-farm-line bg-farm-terminal p-3 font-pixel text-base leading-snug text-farm-terminal-ink"


cardClass : String
cardClass =
    "space-y-4 border-[3px] border-farm-line bg-farm-surface-raised p-4 shadow-pixel sm:p-6"


subCardClass : String
subCardClass =
    "border-2 border-farm-line bg-farm-field p-3"


insetPanelClass : String
insetPanelClass =
    "border-2 border-farm-line-soft bg-farm-surface"
