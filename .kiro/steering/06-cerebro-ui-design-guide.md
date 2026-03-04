---
inclusion: auto
---

# Cerebro UI Design Guide - Modernization Strategy

## Analysis of Current Cerebro UI (Legacy)

Based on the provided screenshots, here's a comprehensive analysis of the current design:

### Color Palette (Current)

**Dark Theme (Primary)**
- Background: Dark charcoal/slate (`#2d3436`, `#353b3f`)
- Secondary background: Slightly lighter dark (`#3d4447`)
- Accent color: Bright cyan/teal (`#00d9ff`, `#1dd1a1`) - used for highlights
- Text primary: White/light gray (`#ffffff`, `#dfe6e9`)
- Text secondary: Medium gray (`#b2bec3`)
- Success/Green: Bright green (`#00b894`, `#55efc4`) - for healthy shards/status
- Warning/Yellow: Yellow (`#fdcb6e`) - for yellow cluster status
- Error/Red: Red (`#d63031`, `#ff7675`) - for errors and high load
- Border color: Subtle dark gray (`#4a5259`)

**Top Navigation Bar**
- Background: Very dark (`#1e272e`)
- Accent stripe: Bright cyan (`#00d9ff`) - 2-3px horizontal line below nav

### Layout Structure (Current)

**1. Top Navigation Bar**
- Height: ~40-50px
- Logo: Left side (flame icon)
- Navigation items: "overview", "rest", "more" dropdown
- Server URL: Center-right
- Settings/logout icons: Far right
- Accent stripe: Bright cyan line below entire nav

**2. Cluster Overview Header**
- Cluster name: Left side, large text
- Key metrics in boxes: nodes, indices, shards, docs, storage size
- Filters: "filter indices by name or all", checkboxes for "closed", "special"
- Pagination: "11-15 of 72" with prev/next arrows

**3. Main Content Area**
- Full width, dark background
- Data presented in grid/table format
- Heavy use of visual indicators (colored boxes for shards)

**4. Shard Allocation Visualization**
- Grid layout showing indices (columns) and nodes (rows)
- Each shard represented as a small colored square:
  - Green: Healthy/assigned shard
  - Cyan outline: Primary shard
  - Gray/dimmed: Unassigned or replica
- Node information on left: name, IP, heap/disk/cpu/load indicators
- Index information on top: name, shard count, doc count, size

**5. Node List View**
- Vertical list of nodes
- Each node shows:
  - Lock icon (if master)
  - Node name (large, bold)
  - IP address (below name)
  - Four metrics with labels: heap, disk, cpu, load
  - Horizontal bars for visual representation (cyan for normal, red for high)

**6. REST Console**
- Three-panel layout:
  - Left sidebar: "previous requests" with history list
  - Center: Request input area (dark background)
  - Right: Response output (JSON with syntax highlighting)
- Input area: Method dropdown (GET) + endpoint field
- Bottom buttons: "cURL", "format", "send" (green)
- JSON syntax highlighting: 
  - Keys: Cyan
  - Strings: Green
  - Booleans: Orange
  - Braces/structure: White

**7. Error Messages**
- Red background with darker red border
- Error icon (triangle with exclamation)
- Error text in white
- Dismissible (X button on right)

### Typography (Current)

- **Primary font**: Sans-serif (likely Arial, Helvetica, or system font)
- **Headings**: Bold, larger size
- **Body text**: Regular weight, medium size
- **Monospace**: Used for IPs, IDs, JSON (likely Courier or Monaco)
- **Font sizes**: Relatively small, compact layout

### Component Patterns (Current)

**Metric Boxes**
- Dark background with subtle border
- Label on top (small, gray)
- Value below (large, white)
- Compact, inline layout

**Filters/Search**
- Dark input fields with light text
- Checkboxes with labels
- Inline layout

**Tables/Grids**
- No visible borders between cells
- Alternating row backgrounds (very subtle)
- Compact row height
- Hover states (likely lighter background)

**Buttons**
- Minimal style, flat design
- Primary action: Green background
- Secondary: Dark background with border
- Icon buttons: Just icon, no background

**Dropdowns**
- Dark background
- Light text
- Chevron icon indicator

### Visual Hierarchy (Current)

1. **Top bar accent** - Bright cyan stripe immediately draws attention
2. **Cluster metrics** - Large numbers in boxes
3. **Shard visualization** - Color-coded grid dominates the view
4. **Node/index names** - Bold, larger text
5. **Secondary info** - Smaller, gray text

### Spacing (Current)

- **Very compact** - Minimal padding and margins
- **Dense information** - Maximizes data visibility
- **Tight grid** - Shards packed closely together
- **Small gaps** - Between sections and components

---

## Modernization Strategy for Cerebro Rewrite

### Goals

1. **Maintain familiarity** - Keep the dark theme, cyan accent, and general layout
2. **Improve readability** - Better spacing, typography, and contrast
3. **Enhance usability** - Clearer interactions, better feedback
4. **Add polish** - Smooth animations, better shadows, modern components
5. **Improve accessibility** - WCAG AA compliance, keyboard navigation
6. **Responsive design** - Work well on different screen sizes

### Updated Color Palette (Mantine-based)

**Dark Theme (Primary)**
```typescript
const theme = createTheme({
  primaryColor: 'cyan',
  colors: {
    dark: [
      '#C1C2C5', // dark.0 - lightest
      '#A6A7AB', // dark.1
      '#909296', // dark.2
      '#5C5F66', // dark.3
      '#373A40', // dark.4
      '#2C2E33', // dark.5
      '#25262B', // dark.6 - main background
      '#1A1B1E', // dark.7 - darker background
      '#141517', // dark.8 - darkest
      '#101113', // dark.9 - nav bar
    ],
    cyan: [
      '#C5F6FA', // cyan.0
      '#99E9F2', // cyan.1
      '#66D9E8', // cyan.2
      '#3BC9DB', // cyan.3
      '#22B8CF', // cyan.4 - main accent
      '#15AABF', // cyan.5
      '#1098AD', // cyan.6
      '#0C8599', // cyan.7
      '#0B7285', // cyan.8
      '#095C6B', // cyan.9
    ],
  },
  defaultRadius: 'md',
  fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif',
  fontFamilyMonospace: 'ui-monospace, SFMono-Regular, "SF Mono", Menlo, Monaco, Consolas, monospace',
});
```

**Status Colors**
- Success: `green.6` - Healthy clusters/shards
- Warning: `yellow.6` - Yellow cluster status
- Error: `red.6` - Errors and critical issues
- Info: `cyan.4` - Informational highlights

### Layout Improvements

**1. Top Navigation (AppShell Header)**
```typescript
<AppShell.Header height={60}>
  {/* Logo + brand */}
  {/* Navigation items */}
  {/* Cluster selector */}
  {/* User menu + theme toggle */}
</AppShell.Header>
```
- Height: 60px (slightly taller for better touch targets)
- Accent stripe: 3px cyan border-bottom
- Better spacing between items
- Sticky positioning

**2. Sidebar (Optional, for navigation)**
```typescript
<AppShell.Navbar width={{ base: 250 }}>
  {/* Navigation links */}
  {/* Collapsible on mobile */}
</AppShell.Navbar>
```
- Width: 250px on desktop
- Collapsible on mobile/tablet
- Smooth slide animation

**3. Main Content Area**
```typescript
<AppShell.Main>
  {/* Page content with proper padding */}
</AppShell.Main>
```
- Padding: 24px (more breathing room)
- Max-width: 1920px (for ultra-wide screens)
- Centered content

### Component Modernization

**Metric Cards (Cluster Stats)**
```typescript
<Card shadow="sm" padding="lg" radius="md">
  <Text size="sm" c="dimmed">Nodes</Text>
  <Text size="xl" fw={700}>6</Text>
</Card>
```
- Use Mantine Card component
- Subtle shadow for depth
- Rounded corners (8px)
- Better padding (16px)
- Hover effect (slight elevation increase)

**Shard Visualization Grid**
- Maintain grid layout but with better spacing
- Larger shard boxes (16x16px → 20x20px)
- Subtle border radius (2px)
- Smooth color transitions
- Tooltip on hover showing shard details
- Better visual distinction between primary/replica

**Node List**
```typescript
<Stack gap="md">
  <Card shadow="sm" padding="md">
    <Group justify="space-between">
      <div>
        <Group gap="xs">
          <IconLock size={16} />
          <Text fw={600}>elasticsearch-data-0</Text>
        </Group>
        <Text size="sm" c="dimmed">10.16.0.88</Text>
      </div>
      <Group gap="lg">
        <MetricBar label="heap" value={45} color="cyan" />
        <MetricBar label="disk" value={23} color="cyan" />
        <MetricBar label="cpu" value={12} color="cyan" />
        <MetricBar label="load" value={78} color="red" />
      </Group>
    </Group>
  </Card>
</Stack>
```
- Card-based layout with shadows
- Better spacing between nodes
- Improved metric bars with labels
- Smooth animations on value changes

**REST Console**
```typescript
<Grid>
  <Grid.Col span={3}>
    <ScrollArea>
      {/* Request history */}
    </ScrollArea>
  </Grid.Col>
  <Grid.Col span={9}>
    <Stack>
      <Group>
        <Select data={['GET', 'POST', 'PUT', 'DELETE']} />
        <TextInput flex={1} placeholder="/_snapshot" />
        <Button>Send</Button>
      </Group>
      <Editor height="300px" language="json" theme="vs-dark" />
      <Editor height="400px" language="json" theme="vs-dark" value={response} readOnly />
    </Stack>
  </Grid.Col>
</Grid>
```
- Use Monaco Editor (already in dependencies)
- Better syntax highlighting
- Resizable panels
- Copy button for responses
- Format button for JSON

**Tables**
```typescript
<Table striped highlightOnHover>
  <Table.Thead>
    <Table.Tr>
      <Table.Th>Index</Table.Th>
      <Table.Th>Health</Table.Th>
      <Table.Th>Docs</Table.Th>
      <Table.Th>Size</Table.Th>
    </Table.Tr>
  </Table.Thead>
  <Table.Tbody>
    {/* rows */}
  </Table.Tbody>
</Table>
```
- Striped rows for better readability
- Hover highlighting
- Sortable columns
- Better cell padding

**Notifications (Errors/Success)**
```typescript
notifications.show({
  title: 'Error',
  message: 'Failed to connect to cluster',
  color: 'red',
  icon: <IconX />,
  autoClose: 5000,
});
```
- Use Mantine Notifications
- Auto-dismiss after 5 seconds
- Dismissible manually
- Stack multiple notifications
- Smooth slide-in animation

### Typography Improvements

**Font Sizes**
- xs: 12px (labels, captions)
- sm: 14px (body text, secondary info)
- md: 16px (primary body text)
- lg: 18px (subheadings)
- xl: 20px (headings, large numbers)
- 2xl: 24px (page titles)

**Font Weights**
- 400: Regular body text
- 500: Medium (emphasized text)
- 600: Semi-bold (subheadings)
- 700: Bold (headings, important numbers)

**Line Heights**
- Increase from current compact layout
- 1.5 for body text
- 1.3 for headings

### Spacing System

**Mantine Spacing Scale**
- xs: 8px
- sm: 12px
- md: 16px
- lg: 24px
- xl: 32px

**Component Spacing**
- Card padding: md (16px)
- Section gaps: lg (24px)
- Grid gaps: md (16px)
- Button padding: sm (12px horizontal, 8px vertical)

### Animations & Transitions

**Smooth Transitions**
```css
transition: all 150ms ease;
```

**Hover Effects**
- Cards: Slight elevation increase (shadow)
- Buttons: Background color change
- Links: Color change
- Shard boxes: Scale up slightly (1.1x)

**Loading States**
- Skeleton loaders for data
- Spinner for actions
- Progress bars for operations

### Accessibility Improvements

**Color Contrast**
- Ensure WCAG AA compliance (4.5:1 for normal text)
- Use Mantine's built-in contrast checking
- Don't rely solely on color for status

**Keyboard Navigation**
- All interactive elements focusable
- Visible focus indicators (cyan outline)
- Keyboard shortcuts for common actions
- Skip links for navigation

**Screen Reader Support**
- Proper ARIA labels
- Semantic HTML
- Announce dynamic content changes
- Descriptive button labels

### Responsive Design

**Breakpoints**
- xs: 0-576px (mobile)
- sm: 576-768px (tablet portrait)
- md: 768-992px (tablet landscape)
- lg: 992-1200px (desktop)
- xl: 1200px+ (large desktop)

**Mobile Adaptations**
- Collapsible sidebar
- Stacked metric cards
- Simplified shard visualization
- Touch-friendly targets (44x44px minimum)
- Horizontal scrolling for wide tables

### Dark/Light Theme Support

**Theme Toggle**
- Icon button in header (sun/moon)
- Persist preference to localStorage
- System preference detection
- Smooth transition between themes

**Light Theme Colors** (for future)
- Background: White/light gray
- Text: Dark gray/black
- Accent: Same cyan (adjusted for contrast)
- Borders: Light gray

---

## Implementation Checklist

### Phase 1: Core Layout
- [ ] AppShell with Header, Navbar, Main
- [ ] Top navigation with accent stripe
- [ ] Theme toggle functionality
- [ ] Responsive breakpoints

### Phase 2: Component Library
- [ ] Metric cards for cluster stats
- [ ] Node list cards
- [ ] Shard visualization grid
- [ ] Data tables with sorting
- [ ] Search/filter inputs
- [ ] Action buttons

### Phase 3: Advanced Features
- [ ] REST console with Monaco Editor
- [ ] Notification system
- [ ] Loading states and skeletons
- [ ] Error boundaries
- [ ] Keyboard shortcuts

### Phase 4: Polish
- [ ] Smooth animations
- [ ] Hover effects
- [ ] Focus indicators
- [ ] Accessibility audit
- [ ] Performance optimization

---

## Key Principles

1. **Familiarity First** - Users should recognize Cerebro immediately
2. **Progressive Enhancement** - Start with current design, add modern touches
3. **Data Density** - Maintain ability to see lots of information at once
4. **Visual Clarity** - Use color, spacing, and typography to guide attention
5. **Performance** - Fast rendering, smooth animations, efficient updates
6. **Accessibility** - Usable by everyone, including keyboard and screen reader users

---

## Mantine Components to Use

- **Layout**: AppShell, Container, Grid, Stack, Group, Flex
- **Data Display**: Table, Card, Badge, Text, Title
- **Inputs**: TextInput, Select, Checkbox, Button
- **Feedback**: Notifications, Loader, Progress, Skeleton
- **Navigation**: Tabs, Breadcrumbs, Pagination
- **Overlays**: Modal, Drawer, Tooltip, Menu
- **Visualization**: Custom components for shards (using Box/div)

---

## Color Usage Guide

**Cyan Accent** - Use for:
- Primary actions (buttons)
- Links and interactive elements
- Active/selected states
- Progress indicators
- Accent stripe in header

**Green** - Use for:
- Healthy status (cluster, shards)
- Success messages
- Positive metrics

**Yellow** - Use for:
- Warning status
- Caution messages
- Moderate metrics

**Red** - Use for:
- Error status
- Critical alerts
- High load/usage metrics

**Gray** - Use for:
- Disabled states
- Secondary text
- Borders and dividers
- Backgrounds

---

## Notes for Implementation

- Keep the current dark theme as default (it's what users expect)
- The cyan accent color is iconic - maintain it prominently
- Shard visualization is the most complex component - plan carefully
- REST console needs Monaco Editor integration
- Consider virtual scrolling for large lists (many indices/nodes)
- Test with real Elasticsearch data to ensure performance
- Maintain the compact, information-dense feel while improving readability
