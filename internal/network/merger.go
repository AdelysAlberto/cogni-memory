package network

import (
	"fmt"
	"strings"

	"github.com/AdelysAlberto/cogni/internal/core"
	"github.com/AdelysAlberto/cogni/internal/storage"
)

// MergeMemories imports incoming memories into the target storage with safe conflict resolution.
// Invariant: NEVER overwrite local modifications destructively. Create a peer fork upon conflict.
func MergeMemories(target *storage.Storage, incoming []core.Memory, sender string) (*SyncResponse, error) {
	resp := &SyncResponse{
		Success: true,
		Details: make([]string, 0),
	}

	if sender == "" {
		sender = "remote-peer"
	}

	for _, inc := range incoming {
		// Clean incoming project and topic key
		pName := inc.ProjectName
		if pName == "" {
			pName = "shared"
		}
		tKey := inc.TopicKey

		var existing *core.Memory
		var err error

		if tKey != "" {
			existing, err = target.GetMemoryByTopicKey(pName, tKey)
		}

		if err != nil || existing == nil {
			// Case 1: Brand new memory -> insert directly
			newMem := &core.Memory{
				ProjectName:      pName,
				Category:         inc.Category,
				Title:            inc.Title,
				TopicKey:         tKey,
				SummarySignature: inc.SummarySignature,
				Tags:             inc.Tags,
			}
			if _, saveErr := target.SaveMemory(newMem); saveErr != nil {
				resp.Details = append(resp.Details, fmt.Sprintf("Error guardando '%s': %v", inc.Title, saveErr))
				continue
			}
			resp.Inserted++
			continue
		}

		// Case 2: Memory already exists locally -> check for content identity
		contentIdentical := strings.TrimSpace(existing.SummarySignature) == strings.TrimSpace(inc.SummarySignature) &&
			strings.TrimSpace(existing.Title) == strings.TrimSpace(inc.Title)

		if contentIdentical {
			// Already in sync, nothing to do
			continue
		}

		// Case 3: Divergence / Conflict!
		// DO NOT overwrite existing local memory. Fork safely as peer variant.
		cleanSenderTag := strings.ToLower(strings.ReplaceAll(sender, " ", "-"))
		forkTopicKey := fmt.Sprintf("%s@peer-%s", tKey, cleanSenderTag)
		forkTitle := fmt.Sprintf("%s (peer: %s)", inc.Title, sender)

		forkTags := inc.Tags
		if !strings.Contains(forkTags, "sync-conflict") {
			if forkTags != "" {
				forkTags += ",sync-conflict"
			} else {
				forkTags = "sync-conflict"
			}
		}

		forkMem := &core.Memory{
			ProjectName:      pName,
			Category:         inc.Category,
			Title:            forkTitle,
			TopicKey:         forkTopicKey,
			SummarySignature: inc.SummarySignature,
			Tags:             forkTags,
		}

		if _, forkErr := target.SaveMemory(forkMem); forkErr != nil {
			resp.Details = append(resp.Details, fmt.Sprintf("Error al bifurcar conflicto en '%s': %v", inc.Title, forkErr))
			continue
		}

		resp.Conflicts++
		resp.Details = append(resp.Details, fmt.Sprintf("Conflicto resguardado: '%s' conservada localmente; versión remota guardada como '%s'", inc.Title, forkTitle))
	}

	resp.Message = fmt.Sprintf("Sincronización completada: %d nuevas, %d conflictos resguardados", resp.Inserted, resp.Conflicts)
	return resp, nil
}
