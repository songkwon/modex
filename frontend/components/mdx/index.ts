import { Note, Info_, Tip, Warning, Danger, Check, CalloutTag } from "./callout";
import { Card, CardGroup, Columns, Tile } from "./card";
import { Tabs, Tab } from "./tabs";
import { Accordion, AccordionGroup, Expandable } from "./accordion";
import { Steps, Step } from "./steps";
import { Pre, CodeGroup } from "./code";
import { Frame, Panel, Update, Banner, Snippet } from "./blocks";
import { ParamField, ResponseField, Field, Response } from "./fields";
import { Tooltip, Badge, Color, IconTag } from "./inline";
import { Tree, Folder, File } from "./tree";
import { Mermaid } from "./mermaid";

// Component map handed to MDX. Tag names mirror Mintlify exactly.
export const mdxComponents = {
  // callouts
  Note,
  Info: Info_,
  Tip,
  Warning,
  Danger,
  Check,
  Callout: CalloutTag,
  // cards / layout
  Card,
  CardGroup,
  Columns,
  Tile,
  Panel,
  Frame,
  // interactive
  Tabs,
  Tab,
  Accordion,
  AccordionGroup,
  Expandable,
  Steps,
  Step,
  CodeGroup,
  // changelog / banners
  Update,
  Banner,
  // reusable content
  Snippet,
  // api fields
  ParamField,
  ResponseField,
  Field,
  Response,
  // inline
  Tooltip,
  Badge,
  Color,
  Icon: IconTag,
  // structure / diagrams
  Tree,
  Folder,
  File,
  Mermaid,
  // html element overrides
  pre: Pre
};
