module Sharecrop.Sprites exposing (pixel, slugs)

{-| Hand-crafted pixel-art collectible sprites rendered with pure CSS.

No image files are used: each sprite is authored as a list of equal-length
rows where every character is a palette key. A single helper turns those rows
plus a per-sprite palette into a CSS grid of colored cells.

@docs pixel, slugs

-}

import Dict exposing (Dict)
import Html exposing (Html)
import Html.Attributes as Attr


{-| The 25 collectible slugs, in canonical order.
-}
slugs : List String
slugs =
    [ "harvest-star"
    , "golden-sickle"
    , "seedling"
    , "sun-token"
    , "rain-drop"
    , "wheat-sheaf"
    , "red-barn"
    , "scarecrow"
    , "honey-pot"
    , "pumpkin"
    , "apple"
    , "carrot"
    , "beehive"
    , "windmill"
    , "tractor"
    , "silver-plow"
    , "golden-egg"
    , "prize-cow"
    , "lucky-clover"
    , "full-moon-harvest"
    , "cornucopia"
    , "first-harvest-trophy"
    , "founders-seed"
    , "rainbow-field"
    , "golden-combine"
    ]


{-| Render the sprite for a slug as a square grid of colored cells, each
`cell` pixels wide and tall.

    pixel "harvest-star" 6

An unknown slug renders a neutral placeholder so it never crashes.

-}
pixel : String -> Int -> Html msg
pixel slug cell =
    case sprite slug of
        Just ( rows, palette ) ->
            spriteFrom cell rows (Dict.fromList palette)

        Nothing ->
            spriteFrom cell placeholderRows (Dict.fromList placeholderPalette)



-- RENDERING


{-| Turn authored rows + a palette into a CSS grid of `cell`-pixel squares.

Each character in `rows` is a palette key; `'.'` (and space) are transparent.
All styling is inline so the module needs no external CSS class.

-}
spriteFrom : Int -> List String -> Dict Char String -> Html msg
spriteFrom cell rows palette =
    let
        columns =
            rows
                |> List.map String.length
                |> List.maximum
                |> Maybe.withDefault 0

        cellPx =
            String.fromInt cell ++ "px"

        cells =
            rows
                |> List.concatMap (paddedChars columns)
                |> List.map (renderCell cellPx palette)
    in
    Html.div
        [ Attr.style "display" "grid"
        , Attr.style "grid-template-columns" ("repeat(" ++ String.fromInt columns ++ ", " ++ cellPx ++ ")")
        , Attr.style "width" (String.fromInt (columns * cell) ++ "px")
        , Attr.style "height" (String.fromInt (List.length rows * cell) ++ "px")
        , Attr.style "image-rendering" "pixelated"
        , Attr.style "line-height" "0"
        ]
        cells


paddedChars : Int -> String -> List Char
paddedChars columns row =
    let
        chars =
            String.toList row
    in
    chars ++ List.repeat (columns - List.length chars) '.'


renderCell : String -> Dict Char String -> Char -> Html msg
renderCell cellPx palette key =
    let
        background =
            case key of
                '.' ->
                    "transparent"

                ' ' ->
                    "transparent"

                _ ->
                    Dict.get key palette |> Maybe.withDefault "transparent"
    in
    Html.div
        [ Attr.style "width" cellPx
        , Attr.style "height" cellPx
        , Attr.style "background-color" background
        ]
        []



-- PALETTE SHADES (shared color vocabulary)


green : String
green =
    "#3a7d1e"


greenDark : String
greenDark =
    "#245a10"


greenLight : String
greenLight =
    "#7bb661"


gold : String
gold =
    "#f2c14e"


goldDark : String
goldDark =
    "#e8a33d"


brown : String
brown =
    "#8a5a2b"


brownDark : String
brownDark =
    "#5e3a1a"


red : String
red =
    "#c0392b"


blue : String
blue =
    "#4a90d9"


sky : String
sky =
    "#b3cf86"


cream : String
cream =
    "#fbf6e3"


ink : String
ink =
    "#2a2118"


white : String
white =
    "#ffffff"


grey : String
grey =
    "#9aa0a6"


greyLight : String
greyLight =
    "#cdd1d6"


pink : String
pink =
    "#e08aa8"


orange : String
orange =
    "#e8772e"


orangeDark : String
orangeDark =
    "#c25618"


amber : String
amber =
    "#d9952b"


moon : String
moon =
    "#e8e6d0"


nightField : String
nightField =
    "#1c2b14"



-- SPRITE LOOKUP


sprite : String -> Maybe ( List String, List ( Char, String ) )
sprite slug =
    case slug of
        "harvest-star" ->
            Just harvestStar

        "golden-sickle" ->
            Just goldenSickle

        "seedling" ->
            Just seedling

        "sun-token" ->
            Just sunToken

        "rain-drop" ->
            Just rainDrop

        "wheat-sheaf" ->
            Just wheatSheaf

        "red-barn" ->
            Just redBarn

        "scarecrow" ->
            Just scarecrow

        "honey-pot" ->
            Just honeyPot

        "pumpkin" ->
            Just pumpkin

        "apple" ->
            Just apple

        "carrot" ->
            Just carrot

        "beehive" ->
            Just beehive

        "windmill" ->
            Just windmill

        "tractor" ->
            Just tractor

        "silver-plow" ->
            Just silverPlow

        "golden-egg" ->
            Just goldenEgg

        "prize-cow" ->
            Just prizeCow

        "lucky-clover" ->
            Just luckyClover

        "full-moon-harvest" ->
            Just fullMoonHarvest

        "cornucopia" ->
            Just cornucopia

        "first-harvest-trophy" ->
            Just firstHarvestTrophy

        "founders-seed" ->
            Just foundersSeed

        "rainbow-field" ->
            Just rainbowField

        "golden-combine" ->
            Just goldenCombine

        _ ->
            Nothing



-- PLACEHOLDER


placeholderPalette : List ( Char, String )
placeholderPalette =
    [ ( 'k', ink ), ( 'w', white ), ( 'g', grey ) ]


placeholderRows : List String
placeholderRows =
    [ "kkkkkkkkkkkk"
    , "kgggggggggk."
    , "kgwwwwwwwgk."
    , "kgwwkkkwwgk."
    , "kgwkkgkkwgk."
    , "kgwwwwkkwgk."
    , "kgwwwkkwwgk."
    , "kgwwkkwwwgk."
    , "kgwwwwwwwgk."
    , "kgwwkkwwwgk."
    , "kgggggggggk."
    , "kkkkkkkkkkkk"
    ]



-- 1. HARVEST STAR


harvestStar : ( List String, List ( Char, String ) )
harvestStar =
    ( [ "......kk........"
      , ".....koak......."
      , ".....koak......."
      , "....koooak....."
      , "kkkkooooakkkk.."
      , ".koooooooooak.."
      , "..koooooooak..."
      , "...kooooooak..."
      , "...koooooak...."
      , "..kooak.kooak.."
      , ".kooak...kooak."
      , ".kak......kak.."
      ]
    , [ ( 'k', ink ), ( 'o', gold ), ( 'a', goldDark ) ]
    )



-- 2. GOLDEN SICKLE


goldenSickle : ( List String, List ( Char, String ) )
goldenSickle =
    ( [ "...kkkkkk..."
      , "..koooooak.."
      , ".kookkkkoak."
      , "kook....koak"
      , "koa......koa"
      , "kk........kk"
      , "...........k"
      , ".......kk..."
      , ".......bbk.."
      , "......kbbk.."
      , "......kbbk.."
      , "......kbbk.."
      ]
    , [ ( 'k', ink ), ( 'o', gold ), ( 'a', goldDark ), ( 'b', brown ) ]
    )



-- 3. SEEDLING


seedling : ( List String, List ( Char, String ) )
seedling =
    ( [ "............"
      , "....k..k...."
      , "...kgk.kgk.."
      , "..kgGgkgGgk."
      , "..kgGgGgGgk."
      , "...kgkGkgk.."
      , ".....kGk...."
      , ".....kGk...."
      , "..bbbbbbbbb."
      , ".bsssssssssb"
      , ".bsbsbsbsbsb"
      , ".bbbbbbbbbb."
      ]
    , [ ( 'k', ink ), ( 'g', greenLight ), ( 'G', green ), ( 'b', brownDark ), ( 's', brown ) ]
    )



-- 4. SUN TOKEN


sunToken : ( List String, List ( Char, String ) )
sunToken =
    ( [ "..r..r..r..r"
      , "...r.r..r.r."
      , "....kkkkk..."
      , "r..koooook.r"
      , "..koooooook."
      , "r.koooooook."
      , "..koooooook."
      , "r.koooooook."
      , "..koooooook."
      , "r..koooook.r"
      , "....kkkkk..."
      , "..r.r.r.r.r."
      ]
    , [ ( 'k', goldDark ), ( 'o', gold ), ( 'r', goldDark ) ]
    )



-- 5. RAIN DROP


rainDrop : ( List String, List ( Char, String ) )
rainDrop =
    ( [ ".....kk....."
      , ".....bb....."
      , "....kbbk...."
      , "....bbbb...."
      , "...kbbwbk..."
      , "...bbbwbb..."
      , "..kbbbbbbk.."
      , "..bbwbbbbb.."
      , "..bbwbbbbb.."
      , "..kbbbbbbk.."
      , "...kbbbbk..."
      , "....kkkk...."
      ]
    , [ ( 'k', ink ), ( 'b', blue ), ( 'w', "#bcdcf5" ) ]
    )



-- 6. WHEAT SHEAF


wheatSheaf : ( List String, List ( Char, String ) )
wheatSheaf =
    ( [ "..o..o..o..."
      , ".oao.oao.oa."
      , ".oao.oao.oa."
      , "..o..o..o..."
      , ".oao.oao.oa."
      , "..o.aoa..o.."
      , "...oaoao...."
      , "...oaoao...."
      , "...bbbbb...."
      , "..bbBBBbb..."
      , "...bbbbb...."
      , "..o.....o.."
      ]
    , [ ( 'o', gold ), ( 'a', goldDark ), ( 'b', brown ), ( 'B', brownDark ) ]
    )



-- 7. RED BARN


redBarn : ( List String, List ( Char, String ) )
redBarn =
    ( [ ".....kk....."
      , "...kkrrkk..."
      , ".kkrrrrrrkk."
      , "krrrrrrrrrrk"
      , "krrrrrrrrrrk"
      , "krwwrrrrwwrk"
      , "krwwrrrrwwrk"
      , "krrrwwwwrrrk"
      , "krrwkkkkwrrk"
      , "krrwkwwkwrrk"
      , "krrwkwwkwrrk"
      , "kkkwkkkkwkkk"
      ]
    , [ ( 'k', ink ), ( 'r', red ), ( 'w', white ) ]
    )



-- 8. SCARECROW


scarecrow : ( List String, List ( Char, String ) )
scarecrow =
    ( [ "....kkkkk..."
      , "...kaaaaak.."
      , "..kkkkkkkkk."
      , "....ooooo..."
      , "...okxoxko.."
      , "...ooooooo.."
      , "...okwwwko.."
      , "..o..ggg..o."
      , "ooooogggooooo"
      , "...kogggok.."
      , "....ggggg..."
      , "....b...b..."
      ]
    , [ ( 'k', ink ), ( 'a', brownDark ), ( 'o', gold ), ( 'x', ink ), ( 'w', white ), ( 'g', red ), ( 'b', brown ) ]
    )



-- 9. HONEY POT


honeyPot : ( List String, List ( Char, String ) )
honeyPot =
    ( [ ".......kk..."
      , ".......bk..."
      , "......bbk..."
      , "....kbbbk..."
      , "...koooook.."
      , "..kkkkkkkk.."
      , "..kaaaaaak.."
      , ".kahhhhhhak."
      , ".kahhhhhhak."
      , ".kahhhhhhak."
      , ".kaahhhhaak."
      , "..kkaaaakk.."
      ]
    , [ ( 'k', ink ), ( 'b', brown ), ( 'o', gold ), ( 'a', amber ), ( 'h', "#f0b84a" ) ]
    )



-- 10. PUMPKIN


pumpkin : ( List String, List ( Char, String ) )
pumpkin =
    ( [ "......gg...."
      , ".....gGg...."
      , ".....gg....."
      , "...kkkkkkk.."
      , "..koaoaoaok."
      , ".koaoaoaoaok"
      , ".koaoaoaoaok"
      , ".koaoaoaoaok"
      , ".koaoaoaoaok"
      , "..koaoaoaok."
      , "...kkkkkkk.."
      , "..........."
      ]
    , [ ( 'k', ink ), ( 'o', orange ), ( 'a', orangeDark ), ( 'g', green ), ( 'G', greenLight ) ]
    )



-- 11. APPLE


apple : ( List String, List ( Char, String ) )
apple =
    ( [ "......b....."
      , ".....b.gg..."
      , ".....b.gGg.."
      , "...kkbkk...."
      , "..krrrwrrk.."
      , ".krrrrrwrrk."
      , ".krrrrrrrrk."
      , ".krrrrrrrrk."
      , ".krrrrrrrrk."
      , "..krrrrrrk.."
      , "..krrrrrrk.."
      , "...kkrrkk..."
      ]
    , [ ( 'k', ink ), ( 'r', red ), ( 'w', "#e88" ), ( 'b', brownDark ), ( 'g', green ), ( 'G', greenLight ) ]
    )



-- 12. CARROT


carrot : ( List String, List ( Char, String ) )
carrot =
    ( [ "...g.g.g...."
      , "..gGgGgGg..."
      , "...gkgkg...."
      , "....kkk....."
      , "...koook...."
      , "...koaok...."
      , "...koook...."
      , "....koak...."
      , "....koak...."
      , ".....kok...."
      , ".....kak...."
      , "......k....."
      ]
    , [ ( 'k', ink ), ( 'o', orange ), ( 'a', orangeDark ), ( 'g', green ), ( 'G', greenLight ) ]
    )



-- 13. BEEHIVE


beehive : ( List String, List ( Char, String ) )
beehive =
    ( [ ".....kk....."
      , "...kkooKK..."
      , "..kooooooak."
      , ".kooooooooak"
      , ".kKKKKKKKKak"
      , ".koooooooook"
      , ".kKKKKKKKKak"
      , ".kooooooooak"
      , ".kKKKkkKKKak"
      , ".koookkooook"
      , ".kKKkkkkKKak"
      , "..kkkkkkkk.."
      ]
    , [ ( 'k', ink ), ( 'o', gold ), ( 'K', goldDark ), ( 'a', brownDark ) ]
    )



-- 14. WINDMILL


windmill : ( List String, List ( Char, String ) )
windmill =
    ( [ "kw......wk.."
      , ".kww..wwk..."
      , "..kwwwwk...."
      , "...kook....."
      , "..wwkkww...."
      , "..wwkkww...."
      , "...kook....."
      , "...kbbk....."
      , "..kbwwbk...."
      , "..kbwwbk...."
      , "..kbwwbk...."
      , "..kkkkkk...."
      ]
    , [ ( 'k', ink ), ( 'w', white ), ( 'o', red ), ( 'b', brown ) ]
    )



-- 15. TRACTOR


tractor : ( List String, List ( Char, String ) )
tractor =
    ( [ "............"
      , "......kkkk.."
      , ".....kgggk.."
      , "..kkkkgggk.."
      , ".kgggkkkkk.."
      , ".kgggggggk.."
      , "kkgggggggkk."
      , "kykkkkkkkyk."
      , "kywkkkkkywk."
      , ".kywkkkywk.."
      , "yk.kywyk.ky."
      , "............"
      ]
    , [ ( 'k', ink ), ( 'g', green ), ( 'y', gold ), ( 'w', greyLight ) ]
    )



-- 16. SILVER PLOW


silverPlow : ( List String, List ( Char, String ) )
silverPlow =
    ( [ "............"
      , "........kk.."
      , ".......kgsk."
      , "......kgssk."
      , ".kk..kgssk.."
      , "kssk.kgsk..."
      , "kgsskgsk...."
      , ".kgsgssk...."
      , "..kgssk....."
      , "...kgsk....."
      , "....kk......"
      , "............"
      ]
    , [ ( 'k', ink ), ( 's', white ), ( 'g', grey ) ]
    )



-- 17. GOLDEN EGG


goldenEgg : ( List String, List ( Char, String ) )
goldenEgg =
    ( [ "....kkkk...."
      , "...kooowk..."
      , "..kooowook.."
      , ".kooowoook.."
      , ".koowooaook."
      , "koowoooaaok."
      , "kooooooaaok."
      , "koooooaaaok."
      , "kooooaaaook."
      , ".kooaaaaok.."
      , ".kkoaaaokk.."
      , "...kkkkk...."
      ]
    , [ ( 'k', ink ), ( 'o', gold ), ( 'a', goldDark ), ( 'w', "#fdeeb0" ) ]
    )



-- 18. PRIZE COW


prizeCow : ( List String, List ( Char, String ) )
prizeCow =
    ( [ "............"
      , ".kk.....kk.."
      , "kwwk...kwwk."
      , "kwwkkkkkwwk."
      , ".kwwwwwwwwk."
      , "kwbbwwwbbwwk"
      , "kwbbwwwbbwwk"
      , "kwwwwpwwwwwk"
      , "kkwwwwwwwwkk"
      , ".kwwkkkkwwk."
      , ".kkk.kk.kkk."
      , "............"
      ]
    , [ ( 'k', ink ), ( 'w', white ), ( 'b', brownDark ), ( 'p', pink ) ]
    )



-- 19. LUCKY CLOVER


luckyClover : ( List String, List ( Char, String ) )
luckyClover =
    ( [ "..kk....kk.."
      , ".kGGk..kGGk."
      , "kGgGGkkGGgGk"
      , "kGGGGGGGGGGk"
      , ".kGGGGGGGGk."
      , "..kkGddGkk.."
      , "...kGddGk..."
      , ".kGGGddGGGk."
      , "kGgGGddGGgGk"
      , "kGGGGddGGGGk"
      , ".kGGkddkGGk."
      , "..kk.dd.kk.."
      ]
    , [ ( 'k', ink ), ( 'G', green ), ( 'g', greenLight ), ( 'd', greenDark ) ]
    )



-- 20. FULL MOON HARVEST


fullMoonHarvest : ( List String, List ( Char, String ) )
fullMoonHarvest =
    ( [ "nnnnnnnnnnnn"
      , "nnnn.mmm.nnn"
      , "nn.mmmMmm.nn"
      , "n.mmmmmmmm.n"
      , "n.mmMmmmmm.n"
      , "n.mmmmmMmm.n"
      , "nn.mmmmmm.nn"
      , "nnn.mmmm.nnn"
      , "nnnnnnnnnnnn"
      , "ffffffffffff"
      , "fFfFfFfFfFfF"
      , "FfFfFfFfFfFf"
      ]
    , [ ( 'n', nightField ), ( 'm', moon ), ( 'M', grey ), ( 'f', greenDark ), ( 'F', green ) ]
    )



-- 21. CORNUCOPIA


cornucopia : ( List String, List ( Char, String ) )
cornucopia =
    ( [ "..........k."
      , "........kbbk"
      , ".......kbbk."
      , "......kbbk.r"
      , "....kbbbkrAr"
      , "...kbbbkoArp"
      , "..kbbbkroArp"
      , ".kbbbkAoArpp"
      , ".kbbkrAoArp."
      , "kbbkroArp..."
      , "kbbkkArp...."
      , ".kkkkkk....."
      ]
    , [ ( 'k', ink ), ( 'b', brown ), ( 'r', red ), ( 'A', orange ), ( 'o', gold ), ( 'p', "#9a4fb0" ) ]
    )



-- 22. FIRST HARVEST TROPHY


firstHarvestTrophy : ( List String, List ( Char, String ) )
firstHarvestTrophy =
    ( [ "kkkkkkkkkkk."
      , "koooooooook."
      , "h koooooook h"
      , "hokooooookoh"
      , "hokooooookoh"
      , ".hokoooookh."
      , "..kooooook.."
      , "...kooook..."
      , "....koak...."
      , "....koak...."
      , "...kooook..."
      , "..kkkkkkkk.."
      ]
    , [ ( 'k', ink ), ( 'o', gold ), ( 'a', goldDark ), ( 'h', goldDark ) ]
    )



-- 23. FOUNDERS SEED


foundersSeed : ( List String, List ( Char, String ) )
foundersSeed =
    ( [ "....hhhh...."
      , "..hhwwwwhh.."
      , ".hwwwwwwwwh."
      , ".hwwkkkkwwh."
      , "hwwkooookwwh"
      , "hwkooaaookwh"
      , "hwkoaaaaokwh"
      , "hwkooaaookwh"
      , "hwwkooookwwh"
      , ".hwwkkkkwwh."
      , ".hhwwwwwwhh."
      , "..hhhhhhhh.."
      ]
    , [ ( 'k', ink ), ( 'o', gold ), ( 'a', goldDark ), ( 'w', "#fdeeb0" ), ( 'h', "#fff6cf" ) ]
    )



-- 24. RAINBOW FIELD


rainbowField : ( List String, List ( Char, String ) )
rainbowField =
    ( [ "...rrrrrr..."
      , "..rooooor..."
      , ".rooyyyoor.."
      , ".oyygggyyo.."
      , "oyggbbbggyo."
      , "yggbbppbbggy"
      , "ggbbp..pbbgg"
      , "gbbp....pbbg"
      , "............"
      , "GGGGGGGGGGGG"
      , "GdGdGdGdGdGd"
      , "dGdGdGdGdGdG"
      ]
    , [ ( 'r', red ), ( 'o', orange ), ( 'y', gold ), ( 'g', green ), ( 'b', blue ), ( 'p', "#9a4fb0" ), ( 'G', green ), ( 'd', greenDark ) ]
    )



-- 25. GOLDEN COMBINE


goldenCombine : ( List String, List ( Char, String ) )
goldenCombine =
    ( [ "............"
      , "ooo.....kkk."
      , "oaao..kkoook"
      , "oaaokkkooook"
      , "oaaooooooook"
      , "kkkkkooooook"
      , "kwwwkkkkkkkk"
      , "kwwwkooooook"
      , "kkkkkkkkkkk."
      , ".kykk.kykk.."
      , "kywwyk kywyk"
      , ".kkk...kkk.."
      ]
    , [ ( 'k', ink ), ( 'o', gold ), ( 'a', goldDark ), ( 'w', greyLight ), ( 'y', goldDark ) ]
    )
