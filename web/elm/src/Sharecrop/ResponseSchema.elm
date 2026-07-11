module Sharecrop.ResponseSchema exposing
    ( FieldInputKind(..)
    , FormField
    , ResponseSchema(..)
    , SchemaField
    , buildPartial
    , buildSubmission
    , formFields
    , parse
    )

{-| The task response schema, decoded from the same wire shape the backend's
`internal/schema` parser reads. The worker-side submit form is built from
this: each top-level object field becomes a typed input instead of making the
worker hand-write the whole response JSON into one textarea.
-}

import Dict exposing (Dict)
import Json.Decode as Decode exposing (Decoder)
import Json.Encode as Encode


type ResponseSchema
    = SchemaObject (List SchemaField)
    | SchemaArray ResponseSchema
    | SchemaString
    | SchemaInteger
    | SchemaDecimalString
    | SchemaEnum (List String)
    | SchemaLiteral String
    | SchemaUnion (List ResponseSchema)
    | SchemaFreeform


type alias SchemaField =
    { name : String
    , required : Bool
    , schema : ResponseSchema
    }


parse : String -> Maybe ResponseSchema
parse raw =
    Decode.decodeString decoder raw |> Result.toMaybe


decoder : Decoder ResponseSchema
decoder =
    Decode.field "kind" Decode.string
        |> Decode.andThen
            (\kind ->
                case kind of
                    "object" ->
                        Decode.map SchemaObject
                            (Decode.field "fields" (Decode.list fieldDecoder))

                    "array" ->
                        Decode.map SchemaArray
                            (Decode.field "item" (Decode.lazy (\_ -> decoder)))

                    "string" ->
                        Decode.succeed SchemaString

                    "integer" ->
                        Decode.succeed SchemaInteger

                    "decimal_string" ->
                        Decode.succeed SchemaDecimalString

                    "enum" ->
                        Decode.map SchemaEnum (Decode.field "values" (Decode.list Decode.string))

                    "literal" ->
                        Decode.map SchemaLiteral (Decode.field "value" Decode.string)

                    "union" ->
                        Decode.map SchemaUnion
                            (Decode.field "variants" (Decode.list (Decode.lazy (\_ -> decoder))))

                    "freeform" ->
                        Decode.succeed SchemaFreeform

                    _ ->
                        Decode.fail ("unsupported schema kind: " ++ kind)
            )


fieldDecoder : Decoder SchemaField
fieldDecoder =
    Decode.map3 SchemaField
        (Decode.field "name" Decode.string)
        (Decode.field "presence" Decode.string |> Decode.map (\presence -> presence == "required"))
        (Decode.field "schema" (Decode.lazy (\_ -> decoder)))


{-| How one top-level object field is edited in the submit form.
-}
type FieldInputKind
    = TextInput
    | IntegerInput
    | DecimalInput
    | EnumSelect (List String)
    | FixedLiteral String
    | LinesArray
    | JsonArrayInput
    | JsonInput


type alias FormField =
    { name : String
    , required : Bool
    , input : FieldInputKind
    }


{-| The typed form only exists for a top-level object schema; every other
top-level shape (freeform, bare array, union...) keeps the raw JSON editor.
-}
formFields : ResponseSchema -> Maybe (List FormField)
formFields schema =
    case schema of
        SchemaObject fields ->
            Just (List.map formField fields)

        _ ->
            Nothing


formField : SchemaField -> FormField
formField field =
    { name = field.name
    , required = field.required
    , input =
        case field.schema of
            SchemaString ->
                TextInput

            SchemaInteger ->
                IntegerInput

            SchemaDecimalString ->
                DecimalInput

            SchemaEnum values ->
                EnumSelect values

            SchemaLiteral value ->
                FixedLiteral value

            SchemaArray SchemaString ->
                LinesArray

            SchemaArray _ ->
                JsonArrayInput

            SchemaObject _ ->
                JsonInput

            SchemaUnion _ ->
                JsonInput

            SchemaFreeform ->
                JsonInput
    }


{-| Build the response JSON from the per-field inputs. Returns the first
problem as an error message instead of submitting something the server (or
the schema validator) will reject as a surprise.
-}
buildSubmission : List FormField -> Dict String String -> Result String String
buildSubmission fields values =
    List.foldl
        (\field acc ->
            case acc of
                Err _ ->
                    acc

                Ok pairs ->
                    case fieldValue field (Dict.get field.name values |> Maybe.withDefault "") of
                        FieldOk value ->
                            Ok (pairs ++ [ ( field.name, value ) ])

                        FieldOmitted ->
                            Ok pairs

                        FieldError message ->
                            Err message
        )
        (Ok [])
        fields
        |> Result.map (\pairs -> Encode.encode 0 (Encode.object pairs))


{-| Build a best-effort JSON object from whatever fields currently have a
usable value, ignoring missing-required and per-field errors. Used to seed
the raw-JSON editor when the worker switches to it, so the values they
already typed are not lost.
-}
buildPartial : List FormField -> Dict String String -> String
buildPartial fields values =
    fields
        |> List.filterMap
            (\field ->
                case fieldValue field (Dict.get field.name values |> Maybe.withDefault "") of
                    FieldOk value ->
                        Just ( field.name, value )

                    FieldOmitted ->
                        Nothing

                    FieldError _ ->
                        Nothing
            )
        |> Encode.object
        |> Encode.encode 2


type FieldOutcome
    = FieldOk Encode.Value
    | FieldOmitted
    | FieldError String


fieldValue : FormField -> String -> FieldOutcome
fieldValue field raw =
    let
        trimmed =
            String.trim raw

        missing =
            trimmed == ""
    in
    case field.input of
        FixedLiteral value ->
            -- Included automatically; there is nothing for the worker to type.
            FieldOk (Encode.string value)

        TextInput ->
            if missing then
                requireOrOmit field

            else
                FieldOk (Encode.string raw)

        DecimalInput ->
            if missing then
                requireOrOmit field

            else
                FieldOk (Encode.string trimmed)

        IntegerInput ->
            if missing then
                requireOrOmit field

            else
                case String.toInt trimmed of
                    Just number ->
                        FieldOk (Encode.int number)

                    Nothing ->
                        FieldError (field.name ++ ": enter a whole number")

        EnumSelect values ->
            if missing then
                requireOrOmit field

            else if List.member trimmed values then
                FieldOk (Encode.string trimmed)

            else
                FieldError (field.name ++ ": choose one of the listed values")

        LinesArray ->
            let
                items =
                    raw
                        |> String.lines
                        |> List.map String.trim
                        |> List.filter (\line -> line /= "")
            in
            if List.isEmpty items then
                requireOrOmit field

            else
                FieldOk (Encode.list Encode.string items)

        JsonArrayInput ->
            if missing then
                requireOrOmit field

            else
                case Decode.decodeString (Decode.list Decode.value) trimmed of
                    Ok items ->
                        FieldOk (Encode.list identity items)

                    Err _ ->
                        FieldError (field.name ++ ": enter a JSON array, e.g. [ ... ]")

        JsonInput ->
            if missing then
                requireOrOmit field

            else
                case Decode.decodeString Decode.value trimmed of
                    Ok value ->
                        FieldOk value

                    Err _ ->
                        FieldError (field.name ++ ": enter valid JSON")


requireOrOmit : FormField -> FieldOutcome
requireOrOmit field =
    if field.required then
        FieldError (field.name ++ ": this field is required")

    else
        FieldOmitted
