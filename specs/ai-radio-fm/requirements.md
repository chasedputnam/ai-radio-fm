# Requirements Document

## Introduction

AI Radio FM (a Golang rewrite of writ-fm) is a 24/7 AI-powered music-forward internet radio station. The system autonomously generates music, writes and speaks short hosted talk breaks, and streams the content continuously. The platform features multiple AI hosts with distinct personalities rotating across a defined schedule of shows, and an autonomous operator loop to maintain content supply and system health. The system will be rewritten in Golang for improved performance, concurrency, and operational simplicity.

## Requirements

### Requirement 1: Streaming and Playback Management
**User Story:** As an operator, I want the system to continuously stream audio, so that listeners can tune in 24/7 without interruption.
#### Acceptance Criteria
- WHEN the streaming service is started THEN the system SHALL encode and stream Ogg Vorbis audio to an Icecast server.
- WHEN a new show block begins THEN the system SHALL build a playlist dynamically based on the current schedule and available content.
- WHEN a playlist completes or is near completion THEN the system SHALL seamlessly enqueue the next block of content.
- IF new content is generated during playback THEN the system SHALL detect it and safely reload the playlist or inject it without dropping the active stream.

### Requirement 2: AI Content Generation (Talk and Music)
**User Story:** As a listener, I want to hear fresh AI-generated music and themed hosted breaks, so that the radio station feels alive and engaging.
#### Acceptance Criteria
- WHEN content generation is triggered THEN the system SHALL use an LLM (e.g., Claude) to generate talk break scripts matching the current host's persona and show topic.
- WHEN a talk script is generated THEN the system SHALL render it into audio using a TTS engine (e.g., Kokoro).
- WHEN music generation is triggered THEN the system SHALL interface with a music generation backend (e.g., music-gen.server) to create thematic music tracks.
- IF content levels for an upcoming show fall below a configurable threshold THEN the system SHALL automatically trigger generation of new talk breaks and music bumpers.

### Requirement 3: Scheduling and Personas
**User Story:** As a station manager, I want to define show schedules and host personalities, so that the station has structured programming and diverse voices.
#### Acceptance Criteria
- WHEN the system initializes THEN it SHALL load a weekly show schedule mapping time slots to specific shows and hosts.
- WHEN a host is assigned to a show THEN the system SHALL use that host's defined persona (voice style, philosophy, prompt rules) for all generated talk segments.
- WHEN the current time crosses a scheduled show boundary THEN the system SHALL transition to the new show's introductory content and switch the active host context.

### Requirement 4: Autonomous Operator Loop
**User Story:** As a maintainer, I want the station to run autonomously, so that I do not have to manually stock content or monitor stream health.
#### Acceptance Criteria
- WHEN the operator daemon is active THEN the system SHALL periodically perform health checks on the stream, API, and Icecast server.
- WHEN the operator loop runs THEN the system SHALL read the station ledger and recent history to maintain editorial continuity across talk segments.
- IF a listener message is received THEN the system SHALL queue it for the listener response generator to produce an on-air reply.
- IF a system component (streamer, encoder) fails THEN the operator daemon SHALL log the error and attempt to restart the component.

### Requirement 5: API and Management CLI
**User Story:** As a developer, I want a single unified CLI and API, so that I can easily manage the station, inspect its status, and expose "now playing" metadata.
#### Acceptance Criteria
- WHEN the API is queried at the `/now-playing` endpoint THEN the system SHALL return the current track, active show, and host metadata.
- WHEN the API is queried at the `/schedule` endpoint THEN the system SHALL return the upcoming programming schedule.
- WHEN the administrator runs the management CLI (e.g., `airadio start`) THEN the system SHALL manage the underlying processes (streaming, API, operator loop) safely.
- WHEN the administrator runs a CLI generation command (e.g., `airadio generate talk`) THEN the system SHALL manually execute the content generation workflow for the specified type.

### Requirement 6: Stream Archival
**User Story:** As an archivist, I want the system to record the continuous audio stream, so that past broadcasts can be saved for historical archiving or on-demand playback.
#### Acceptance Criteria
- WHEN the streaming service is active THEN the system SHALL optionally save a continuous copy of the audio stream to local disk.
- WHEN archival is enabled THEN the system SHALL segment the recorded audio files by time (e.g., hourly or daily) or show blocks to prevent indefinitely large files.
- WHEN an archival segment completes THEN the system SHALL save it with metadata reflecting the date, time, and show.