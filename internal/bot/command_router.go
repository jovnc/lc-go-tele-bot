package bot

import "context"

func (s *Service) handleCommand(ctx context.Context, chatID int64, text string) error {
	return s.commandHandler.Handle(ctx, chatID, text)
}
