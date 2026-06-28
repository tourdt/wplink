# wxapp v3 UI Design

## Goal

Bring the wxapp UI up to the v3 prototype standard for the launch-review path. The work focuses on visual hierarchy, trust signals, resource judgment details, lifecycle states, empty states, and action clarity while preserving the current routes, API contracts, and business behavior.

## Scope

The implementation covers the nine v3 launch-review flows:

- `pages/home/index`
- `pages/search/index`
- `pages/resource/detail`
- `pages/merchant/detail`
- `pages/publish/index`
- `pages/messages/index`
- `pages/my-resources/index`
- `pages/topic/index`
- `pages/webview/index`

Out of scope for this phase:

- Backend API changes
- New database fields
- Authentication flow changes
- Full redesign of secondary pages such as verification, favorites, demand success, publish success, profile, and my center
- Replacing uni-app navigation or tab structure

## Design Direction

The product remains a B2B apparel resource marketplace. The UI should be search-first, resource-first, and trust-first.

Use the v3 prototype as the primary visual reference:

- White cards on a light gray background
- `#0f766e` as the primary trust/action color
- Supporting tones for inventory, factory, activity, and promotion states
- 8px equivalent radius in prototype terms, mapped to compact `rpx` radii in wxapp
- Dense but readable marketplace layout, not a marketing landing page
- Platform trust labels and operational guidance close to the decision point

## Shared UI System

Update shared styling through `wxapp/uni.scss` and existing shared components where practical.

Shared primitives:

- Page background and safe content padding
- Card surfaces
- Primary, secondary, ghost, and warning buttons
- Tags for verified, promoted, status, and neutral metadata
- Section headers
- Empty states
- Sticky bottom action bars
- Resource-card spacing, image ratio, metadata, price, merchant, and refresh-time treatment

The shared system should reduce repeated CSS in the nine pages without introducing a broad component library.

## Page Designs

### Home

Home should mirror the v3 first screen:

- Search entry is prominent and immediately reachable.
- Banner area supports topic, activity webview, and merchant/resource jumps.
- Trust strip shows platform verification, certified merchants, and expiration rules.
- Activity entry can jump to webview.
- Weekly focus card highlights urgent inventory.
- Four scenario cards cover stock, clearance, factory, and publishing.
- Recommended resources use the improved shared resource card.

### Search

Search should help users quickly decide whether a result is worth opening:

- Keep keyword search and type filters.
- Preserve saved search behavior for logged-in users.
- Hot keywords remain visible.
- Show promoted-resource explanation when results exist.
- Result cards expose trust labels, type, quantity, price, refresh time, merchant, and decision hints where existing data allows.
- Empty state naturally routes to purchase demand submission.

### Resource Detail

Detail page should become the strongest decision screen:

- Show a gallery area from cover/images, with a neutral placeholder when images are missing.
- Put verification, status, and freshness tags above the title.
- Show price and key specs before long description.
- Show merchant summary as a tappable trust block.
- Keep contact-before-action guidance near the contact actions.
- Keep same-category recommendations.
- Bottom sticky bar keeps favorite, merchant, phone, wechat, and share actions reachable.

### Merchant Detail

Merchant page should support buyer trust evaluation:

- Header shows merchant identity, type, verification, follow action, and location/category context when available.
- Stats summarize current publishing, historical publishing, and deal/contact signals from existing response fields when available.
- Main categories and credit tags are prominent.
- Rights/promotion note explains certification and top placement without overclaiming.
- Published resource list uses the shared resource card.
- Contact buttons remain guidance-only unless entering from a resource detail provides contact data.

### Publish

Publish page should feel like a guided merchant workflow:

- Top quota/rights card explains publishing and top-voucher value.
- Type switch remains easy to scan horizontally.
- Form groups are visually separated and use clear labels/placeholders.
- Required fields drive the submit-readiness message.
- Image upload shows uploaded image references or thumbnails where available.
- Save draft and submit review remain the two main actions.

### Messages

Messages should cover v3 launch states:

- Tabs retain all, unread, read, audit, and effect categories.
- Message cards emphasize unread state, title, time, target hint, and content.
- Effect card summarizes exposure, views, and contact performance.
- Guidance points merchants to refresh, top, or manage resources.

### My Resources

My resources should make lifecycle and actions clear:

- Header explains status, performance, and promotion management.
- Filter row covers draft, pending, published, expiring soon, expired, dealt, and taken down.
- Each card shows status, expiry, title, category/type, publish and expiry dates, metrics, and allowed actions.
- Destructive or lifecycle actions should stay visually secondary to review/detail actions.

### Topic

Topic page should match v3 banner-topic behavior:

- Hero section describes the configured topic.
- Summary metrics can use existing query result count or static UI copy when no API data exposes counts.
- Filters guide users through urgent clearance, sample-ready, size, and verified states.
- Resource list uses shared resource cards.
- Empty or insufficient results route users to demand submission.

### Webview

Webview page should provide a launch-review wrapper around configured activity URLs:

- Show the decoded URL or a safe short display.
- Show activity cover and copy where route params or fallback copy are available.
- Explain that the URL must belong to an allowed business domain.
- Provide a path back to related platform resources.

## Data And Error Handling

No API contract changes are planned. UI should degrade gracefully:

- Missing images use styled placeholders.
- Missing merchant fields show neutral fallback copy.
- Failed optional calls should not block search, browsing, or publishing.
- Empty lists should show action-oriented empty states.
- Contact actions continue to use existing privacy behavior and metric recording.

## Validation

Implementation is acceptable when:

- The nine scoped pages compile under the existing wxapp toolchain.
- Existing route paths and API calls remain intact.
- The current validation scripts pass where applicable.
- Home can route into search, topic, webview, publish, and resource detail paths.
- Search no-result state routes to demand submission.
- Resource detail contact actions still record contact behavior.
- My resources retains all existing lifecycle actions.
- UI text fits mobile width without horizontal scrolling.

## Implementation Notes

Keep edits scoped and incremental:

- Start with shared styles and `ResourceCard`.
- Update high-traffic pages first: home, search, resource detail.
- Then update merchant, publish, messages, and my resources.
- Finish with topic and webview.
- Do not refactor unrelated business logic while improving UI.
