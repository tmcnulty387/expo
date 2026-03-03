# EXPO
*Team Members: Sebastian LaVine, Jane Majewski, Tim McNulty, Rina Peshori*

## Project Goal:
Create a native, distributed digital whiteboard application.

## Use Cases:
- Shared Note taking
- Collaborative diagramming
- Viewing & editing drawings from multiple devices
- Sharing text and images
- Wireframe design

## Intended Components:
- Gio UI
  - Used for rendering
- Standard Go TCP/IP stack
  - Used for P2P network implementation
- User input receptors
  - Update Queue
    - Update queue has 2 input sources: local user input & network-connected peers
  - Uses protocol/structures that allow us to send a diff rather than the full whiteboard upon update
    - Works with Gio input libraries to receive and restructure data as per our needs
  - Consideration: conflict handling
    - Layer Stack
      - Each user owns (has edit access to) only one layer
      - All layers are displayed across all user devices
- Drawing structures
  - Contains pixel point data (with features such as color value)
- Encryption schema
  - Possibly implemented via HTTPs

TODO: stub out key functions & key data structures

## Testing Goals:
Primary goal: It works :)
- Tested using automated unit tests
- Network failures handled (resiliency)
- Security, authentication
- Latency

## MVP:
- Bitmap image data communicated in real time between 2 clients
- Pixel-based drawing tools only, implemented using bitmap

## Stretch Goals:
- P2P networking
- Text rendering
- Embedded images
- Transforms/scaling
- Export whiteboard as image/other file format
- Web interface

## Expected Checkpoint Functionality
We expect to have all basic functionality as detailed in the MVP section completed by the Checkpoint on April 5.

## Research/Notes
- golibp2p: library for P2P networking in Go