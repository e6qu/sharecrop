module Main exposing (main)

import Browser
import Html exposing (Html, div, h1, main_, p, text)
import Html.Attributes exposing (class)
import Sharecrop.Generated.Auth as Auth


type alias Model =
    { subjectKind : Auth.SubjectKind
    }


type Msg
    = NoOp


main : Program () Model Msg
main =
    Browser.element
        { init = \_ -> ( { subjectKind = Auth.SubjectKindGuest }, Cmd.none )
        , update = update
        , subscriptions = \_ -> Sub.none
        , view = view
        }


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        NoOp ->
            ( model, Cmd.none )


view : Model -> Html Msg
view _ =
    main_ [ class "min-h-screen bg-slate-50 p-8 text-slate-950" ]
        [ div [ class "mx-auto max-w-5xl" ]
            [ h1 [ class "text-3xl font-semibold" ] [ text "Sharecrop" ]
            , p [ class "mt-2 text-slate-600" ] [ text "Project skeleton is running." ]
            ]
        ]
