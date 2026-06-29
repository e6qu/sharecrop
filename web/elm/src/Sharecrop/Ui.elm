module Sharecrop.Ui exposing (..)

import Html exposing (Attribute, Html, button, h1, h2, input, label, p, pre, span, text, textarea)
import Html.Attributes exposing (attribute, class, type_)


testId : String -> Attribute msg
testId value =
    attribute "data-testid" value


card : List (Html msg) -> Html msg
card children =
    Html.div [ class "space-y-4 rounded-lg border border-slate-200 bg-white p-4 shadow-sm sm:p-6" ] children


sectionTitle : String -> Html msg
sectionTitle title =
    h2 [ class "text-lg font-medium" ] [ text title ]


pageTitle : String -> Html msg
pageTitle title =
    h1 [ class "text-3xl font-semibold" ] [ text title ]


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


textarea_ : List (Attribute msg) -> Html msg
textarea_ attrs =
    textarea (class textareaClass :: attrs) []


badge : String -> Html msg
badge value =
    span [ class "inline-flex items-center rounded-full bg-slate-100 px-2.5 py-0.5 text-xs font-medium text-slate-700" ] [ text value ]


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
    "rounded-md border border-red-300 px-4 py-2 text-sm font-medium text-red-700 hover:bg-red-50"


fieldClass : String
fieldClass =
    "w-full min-h-[44px] rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-slate-500 focus:outline-none"


checkboxClass : String
checkboxClass =
    "h-4 w-4 rounded border-slate-400 text-slate-900 focus:ring-2 focus:ring-slate-500"


textareaClass : String
textareaClass =
    "w-full rounded-md border border-slate-300 px-3 py-2 font-mono text-sm focus:border-slate-500 focus:outline-none"


codeBlockClass : String
codeBlockClass =
    "whitespace-pre-wrap break-words rounded-md bg-slate-900 p-3 text-xs text-slate-100"
