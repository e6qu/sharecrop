package assets

// CatalogEntry is one of the platform's default collectibles — a reusable
// template an admin can award. Awarding mints a fresh copy owned by the
// recipient, so many holders can carry the same default. The Art slug names the
// hand-crafted pixel sprite the client renders (see Sharecrop.Sprites).
type CatalogEntry struct {
	Slug   string
	Name   string
	Kind   CollectibleKind
	Policy TransferPolicy
	Art    string
}

// defaultCatalog is the fixed set of 25 default collectibles, themed around the
// harvest/farming arcade aesthetic. Kind doubles as rarity: badge = common,
// edition = rare, unique = legendary. All default to transferable-between-users
// so awarded copies can be traded.
var defaultCatalog = []CatalogEntry{
	// Badges — common.
	{Slug: "harvest-star", Name: "Harvest Star", Kind: CollectibleKindBadge, Policy: TransferPolicyTransferableBetweenUsers, Art: "harvest-star"},
	{Slug: "golden-sickle", Name: "Golden Sickle", Kind: CollectibleKindBadge, Policy: TransferPolicyTransferableBetweenUsers, Art: "golden-sickle"},
	{Slug: "seedling", Name: "Seedling", Kind: CollectibleKindBadge, Policy: TransferPolicyTransferableBetweenUsers, Art: "seedling"},
	{Slug: "sun-token", Name: "Sun Token", Kind: CollectibleKindBadge, Policy: TransferPolicyTransferableBetweenUsers, Art: "sun-token"},
	{Slug: "rain-drop", Name: "Rain Drop", Kind: CollectibleKindBadge, Policy: TransferPolicyTransferableBetweenUsers, Art: "rain-drop"},
	{Slug: "wheat-sheaf", Name: "Wheat Sheaf", Kind: CollectibleKindBadge, Policy: TransferPolicyTransferableBetweenUsers, Art: "wheat-sheaf"},
	{Slug: "red-barn", Name: "Red Barn", Kind: CollectibleKindBadge, Policy: TransferPolicyTransferableBetweenUsers, Art: "red-barn"},
	{Slug: "scarecrow", Name: "Scarecrow", Kind: CollectibleKindBadge, Policy: TransferPolicyTransferableBetweenUsers, Art: "scarecrow"},
	{Slug: "honey-pot", Name: "Honey Pot", Kind: CollectibleKindBadge, Policy: TransferPolicyTransferableBetweenUsers, Art: "honey-pot"},
	{Slug: "pumpkin", Name: "Pumpkin", Kind: CollectibleKindBadge, Policy: TransferPolicyTransferableBetweenUsers, Art: "pumpkin"},
	{Slug: "apple", Name: "Apple", Kind: CollectibleKindBadge, Policy: TransferPolicyTransferableBetweenUsers, Art: "apple"},
	{Slug: "carrot", Name: "Carrot", Kind: CollectibleKindBadge, Policy: TransferPolicyTransferableBetweenUsers, Art: "carrot"},
	{Slug: "beehive", Name: "Beehive", Kind: CollectibleKindBadge, Policy: TransferPolicyTransferableBetweenUsers, Art: "beehive"},
	{Slug: "windmill", Name: "Windmill", Kind: CollectibleKindBadge, Policy: TransferPolicyTransferableBetweenUsers, Art: "windmill"},
	{Slug: "tractor", Name: "Tractor", Kind: CollectibleKindBadge, Policy: TransferPolicyTransferableBetweenUsers, Art: "tractor"},
	// Editions — rare.
	{Slug: "silver-plow", Name: "Silver Plow", Kind: CollectibleKindEdition, Policy: TransferPolicyTransferableBetweenUsers, Art: "silver-plow"},
	{Slug: "golden-egg", Name: "Golden Egg", Kind: CollectibleKindEdition, Policy: TransferPolicyTransferableBetweenUsers, Art: "golden-egg"},
	{Slug: "prize-cow", Name: "Prize Cow", Kind: CollectibleKindEdition, Policy: TransferPolicyTransferableBetweenUsers, Art: "prize-cow"},
	{Slug: "lucky-clover", Name: "Lucky Clover", Kind: CollectibleKindEdition, Policy: TransferPolicyTransferableBetweenUsers, Art: "lucky-clover"},
	{Slug: "full-moon-harvest", Name: "Full-Moon Harvest", Kind: CollectibleKindEdition, Policy: TransferPolicyTransferableBetweenUsers, Art: "full-moon-harvest"},
	// Unique — legendary.
	{Slug: "cornucopia", Name: "Cornucopia", Kind: CollectibleKindUnique, Policy: TransferPolicyTransferableBetweenUsers, Art: "cornucopia"},
	{Slug: "first-harvest-trophy", Name: "First-Harvest Trophy", Kind: CollectibleKindUnique, Policy: TransferPolicyTransferableBetweenUsers, Art: "first-harvest-trophy"},
	{Slug: "founders-seed", Name: "Founder's Seed", Kind: CollectibleKindUnique, Policy: TransferPolicyTransferableBetweenUsers, Art: "founders-seed"},
	{Slug: "rainbow-field", Name: "Rainbow Field", Kind: CollectibleKindUnique, Policy: TransferPolicyTransferableBetweenUsers, Art: "rainbow-field"},
	{Slug: "golden-combine", Name: "Golden Combine", Kind: CollectibleKindUnique, Policy: TransferPolicyTransferableBetweenUsers, Art: "golden-combine"},
}

// Catalog returns the default collectible templates.
func Catalog() []CatalogEntry {
	entries := make([]CatalogEntry, len(defaultCatalog))
	copy(entries, defaultCatalog)
	return entries
}

// CatalogBySlug looks up a default collectible template by its slug.
func CatalogBySlug(slug string) (CatalogEntry, bool) {
	for _, entry := range defaultCatalog {
		if entry.Slug == slug {
			return entry, true
		}
	}
	return CatalogEntry{}, false
}
