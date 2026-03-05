package positouchdriver

import (
	"time"

	"github.com/badpanda83/POSitouch-Integration/entities"
	"github.com/badpanda83/POSitouch-Integration/positouch"
)

// SyncTickets reads all open and closed tickets from the POSitouch XML directories
// and returns them as canonical entities.Ticket values.
func (d *Driver) SyncTickets() ([]entities.Ticket, error) {
	raw, err := positouch.ReadAllTickets(d.cfg.XMLDir, d.cfg.XMLCloseDir)
	if err != nil {
		return nil, err
	}
	return mapTickets(raw), nil
}

func mapTickets(src []positouch.Ticket) []entities.Ticket {
	out := make([]entities.Ticket, 0, len(src))
	for _, t := range src {
		out = append(out, mapTicket(t))
	}
	return out
}

func mapTicket(t positouch.Ticket) entities.Ticket {
	items := make([]entities.TicketItem, 0, len(t.Items))
	for _, it := range t.Items {
		items = append(items, mapTicketItem(it))
	}
	return entities.Ticket{
		Number:         t.Number,
		OpenedAt:       t.OpenedAt.Format(time.RFC3339),
		Table:          t.Table,
		Total:          t.Total,
		ServerName:     t.ServerName,
		PartySize:      t.PartySize,
		CostCenter:     t.CostCenter,
		CostCenterName: t.CostCenterName,
		Items:          items,
		Open:           t.Open,
		POSType:        "positouch",
	}
}

func mapTicketItem(it positouch.Item) entities.TicketItem {
	var opt *entities.TicketItemOption
	if it.Option != nil {
		opt = &entities.TicketItemOption{
			ItemNumber:   it.Option.ItemNumber,
			ItemName:     it.Option.ItemName,
			CellName:     it.Option.CellName,
			Memo:         it.Option.Memo,
			MajorName:    it.Option.MajorName,
			MajorNumber:  it.Option.MajorNumber,
			MinorName:    it.Option.MinorName,
			MinorNumber:  it.Option.MinorNumber,
			ScreenNumber: it.Option.ScreenNumber,
			CellNumber:   it.Option.CellNumber,
		}
	}
	return entities.TicketItem{
		ItemNumber:       it.ItemNumber,
		ItemName:         it.ItemName,
		CellName:         it.CellName,
		MajorName:        it.MajorName,
		MajorNumber:      it.MajorNumber,
		MinorName:        it.MinorName,
		MinorNumber:      it.MinorNumber,
		ScreenNumber:     it.ScreenNumber,
		CellNumber:       it.CellNumber,
		FullPrice:        it.FullPrice,
		NetPrice:         it.NetPrice,
		PriceLevel:       it.PriceLevel,
		SendTime:         it.SendTime,
		SerialNumber:     it.SerialNumber,
		PrepSequence:     it.PrepSequence,
		PrepSequenceName: it.PrepSequenceName,
		SplitItem:        it.SplitItem,
		SentToPrep:       it.SentToPrep,
		NewServer:        it.NewServer,
		NewServerName:    it.NewServerName,
		Option:           opt,
	}
}
