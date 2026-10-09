package service

import (
	"context"
	"github.com/google/uuid"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/util"
	"sort"
	"time"
)

func (s *FeeService) generateYearlyMembershipFees(ctx context.Context, year int, children []domain.Child) (*GenerateResult, error) {
	result := &GenerateResult{}
	dueDate := time.Date(year, 3, 31, 0, 0, 0, 0, time.UTC)
	// The annual membership fee follows the schedule in effect on 1 January.
	schedule, err := s.ScheduleAt(ctx, time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		return nil, err
	}
	membershipFee := schedule.Config.AnnualMembershipFee

	groupedByHousehold := make(map[uuid.UUID][]domain.Child)
	childrenWithoutHousehold := make([]domain.Child, 0)
	for _, child := range children {
		if child.EntryDate.Year() > year {
			continue
		}
		if child.HouseholdID == nil {
			childrenWithoutHousehold = append(childrenWithoutHousehold, child)
			continue
		}
		groupedByHousehold[*child.HouseholdID] = append(groupedByHousehold[*child.HouseholdID], child)
	}

	for householdID, householdChildren := range groupedByHousehold {
		created, skipped, err := s.generateHouseholdMembershipFees(ctx, householdID, householdChildren, year, membershipFee, dueDate)
		if err != nil {
			return nil, err
		}
		result.Created += created
		result.Skipped += skipped

		if err := s.ensureHouseholdMembershipAssignment(ctx, householdID); err != nil {
			return nil, err
		}
	}

	// Legacy fallback: if no household linkage exists, keep per-child generation.
	for _, child := range childrenWithoutHousehold {
		created, err := s.createFeeIfNotExists(ctx, child.ID, nil, domain.FeeTypeMembership, year, nil, membershipFee, dueDate)
		if err != nil {
			return nil, err
		}
		if created {
			result.Created++
		} else {
			result.Skipped++
		}
	}

	return result, nil
}

// generateHouseholdMembershipFees ensures one MEMBERSHIP fee per club member of
// the household for the year. Households without known members keep a single
// household-level fee. Existing fees without a member are adopted by members in
// member-number order before new ones are created; each new fee goes to the
// oldest household child that has no membership fee for the year yet, so each
// member's fee matches payments made with "their" child's number.
func (s *FeeService) generateHouseholdMembershipFees(ctx context.Context, householdID uuid.UUID, householdChildren []domain.Child, year int, amount float64, dueDate time.Time) (created, skipped int, err error) {
	existing, err := s.feeRepo.ListMembershipForHousehold(ctx, householdID, year)
	if err != nil {
		return 0, 0, err
	}
	members, err := s.householdRepo.GetMembersForYear(ctx, householdID, year)
	if err != nil {
		return 0, 0, err
	}

	if len(members) == 0 {
		if len(existing) > 0 {
			return 0, 1, nil
		}
		child := pickRepresentativeChildForMembership(householdChildren)
		if err := s.createMembershipFee(ctx, child.ID, householdID, nil, year, amount, dueDate); err != nil {
			return 0, 0, err
		}
		return 1, 0, nil
	}

	assigned := make(map[uuid.UUID]bool, len(existing))
	usedChildren := make(map[uuid.UUID]bool, len(existing))
	unassigned := make([]domain.FeeExpectation, 0, len(existing))
	for _, fee := range existing {
		usedChildren[fee.ChildID] = true
		if fee.MemberID != nil {
			assigned[*fee.MemberID] = true
		} else {
			unassigned = append(unassigned, fee)
		}
	}

	candidates := sortChildrenByEntry(householdChildren)
	for _, member := range members {
		if assigned[member.ID] {
			skipped++
			continue
		}
		if len(unassigned) > 0 {
			if err := s.feeRepo.AssignMember(ctx, unassigned[0].ID, member.ID); err != nil {
				return 0, 0, err
			}
			unassigned = unassigned[1:]
			skipped++
			continue
		}

		child := candidates[0]
		for _, candidate := range candidates {
			if !usedChildren[candidate.ID] {
				child = candidate
				break
			}
		}
		memberID := member.ID
		if err := s.createMembershipFee(ctx, child.ID, householdID, &memberID, year, amount, dueDate); err != nil {
			return 0, 0, err
		}
		usedChildren[child.ID] = true
		created++
	}

	return created, skipped, nil
}

func (s *FeeService) createMembershipFee(ctx context.Context, childID, householdID uuid.UUID, memberID *uuid.UUID, year int, amount float64, dueDate time.Time) error {
	return s.feeRepo.Create(ctx, &domain.FeeExpectation{
		ID:          uuid.New(),
		ChildID:     childID,
		HouseholdID: &householdID,
		MemberID:    memberID,
		FeeType:     domain.FeeTypeMembership,
		Year:        year,
		Amount:      amount,
		DueDate:     dueDate,
		CreatedAt:   util.Now(),
	})
}

// sortChildrenByEntry returns a copy ordered by entry date (oldest first).
func sortChildrenByEntry(children []domain.Child) []domain.Child {
	sorted := append([]domain.Child(nil), children...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].EntryDate.Equal(sorted[j].EntryDate) {
			return sorted[i].ID.String() < sorted[j].ID.String()
		}
		return sorted[i].EntryDate.Before(sorted[j].EntryDate)
	})
	return sorted
}

func pickRepresentativeChildForMembership(children []domain.Child) domain.Child {
	return sortChildrenByEntry(children)[0]
}

func (s *FeeService) ensureHouseholdMembershipAssignment(ctx context.Context, householdID uuid.UUID) error {
	household, err := s.householdRepo.GetByID(ctx, householdID)
	if err != nil {
		return nil
	}
	if household.MembershipParentID != nil && household.MembershipStatus != "" {
		return nil
	}

	parents, err := s.householdRepo.GetParents(ctx, householdID)
	if err != nil || len(parents) == 0 {
		return nil
	}

	parentID, status := pickHouseholdMembershipParent(parents)
	household.MembershipParentID = &parentID
	household.MembershipStatus = status
	return s.householdRepo.Update(ctx, household)
}

func pickHouseholdMembershipParent(parents []domain.Parent) (uuid.UUID, domain.MembershipAssignmentStatus) {
	candidates := make([]domain.Parent, 0, len(parents))
	for _, parent := range parents {
		if parent.MemberID != nil {
			candidates = append(candidates, parent)
		}
	}

	status := domain.MembershipAssignmentStatusAssumed
	if len(candidates) > 0 {
		parents = candidates
		status = domain.MembershipAssignmentStatusConfirmed
	}

	sort.Slice(parents, func(i, j int) bool {
		if parents[i].CreatedAt.Equal(parents[j].CreatedAt) {
			return parents[i].ID.String() < parents[j].ID.String()
		}
		return parents[i].CreatedAt.Before(parents[j].CreatedAt)
	})

	return parents[0].ID, status
}
