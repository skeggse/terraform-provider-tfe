package tfe

import (
	"errors"
	"fmt"
	"math/rand"
	"testing"
	"time"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestPackTeamOrganizationMemberID(t *testing.T) {
	cases := []struct {
		team                     string
		organizationMembershipID string
		id                       string
	}{
		{
			team:                     "team-47qC3LmA47piVan7",
			organizationMembershipID: "ou-123",
			id:                       "team-47qC3LmA47piVan7/ou-123",
		},
	}

	for _, tc := range cases {
		id := packTeamOrganizationMemberID(tc.team, tc.organizationMembershipID)

		if tc.id != id {
			t.Fatalf("expected ID %q, got %q", tc.id, id)
		}
	}
}

func TestUnpackTeamOrganizationMemberID(t *testing.T) {
	cases := []struct {
		id                       string
		team                     string
		organizationMembershipID string
		err                      bool
	}{
		{
			id:                       "team-47qC3LmA47piVan7/ou-123",
			team:                     "team-47qC3LmA47piVan7",
			organizationMembershipID: "ou-123",
			err:                      false,
		},
		{
			id:                       "some-invalid-id",
			team:                     "",
			organizationMembershipID: "",
			err:                      true,
		},
	}

	for _, tc := range cases {
		team, organizationMembershipID, err := unpackTeamOrganizationMemberID(tc.id)
		if (err != nil) != tc.err {
			t.Fatalf("expected error is %t, got %v", tc.err, err)
		}

		if tc.team != team {
			t.Fatalf("expected team %q, got %q", tc.team, team)
		}

		if tc.organizationMembershipID != organizationMembershipID {
			t.Fatalf("expected organizationMembershipID %q, got %q", tc.organizationMembershipID, organizationMembershipID)
		}
	}
}

func TestAccTFETeamOrganizationMember_basic(t *testing.T) {
	organizationMembership := &tfe.OrganizationMembership{ID: "sauce"}
	rInt := rand.New(rand.NewSource(time.Now().UnixNano())).Int()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckTFETeamOrganizationMemberDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccTFETeamOrganizationMember_basic(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTFETeamOrganizationMemberExists(
						"tfe_team_organization_member.foobar", organizationMembership),
					testAccCheckTFETeamOrganizationMemberAttributes(organizationMembership),
				),
			},
		},
	})
}

func TestAccTFETeamOrganizationMember_import(t *testing.T) {
	rInt := rand.New(rand.NewSource(time.Now().UnixNano())).Int()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckTFETeamOrganizationMemberDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccTFETeamOrganizationMember_basic(rInt),
			},

			{
				ResourceName:      "tfe_team_organization_member.foobar",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckTFETeamOrganizationMemberExists(
	n string, organizationMembership *tfe.OrganizationMembership) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		tfeClient := testAccProvider.Meta().(*tfe.Client)

		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No instance ID is set")
		}

		// Get the team ID and organization membership id.
		teamID, organizationMembershipID, err := unpackTeamOrganizationMemberID(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error unpacking team organization member ID: %w", err)
		}

		organizationMemberships, err := tfeClient.TeamMembers.ListOrganizationMemberships(ctx, teamID)
		if errors.Is(err, tfe.ErrResourceNotFound) {
			return err
		}

		found := false
		for _, om := range organizationMemberships {
			if om.ID == organizationMembershipID {
				found = true
				*organizationMembership = *om
				break
			}
		}

		if !found {
			return fmt.Errorf("Organization membership not found")
		}

		return nil
	}
}

func testAccCheckTFETeamOrganizationMemberAttributes(
	organizationMembership *tfe.OrganizationMembership) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if organizationMembership.Email != "foo@foobar.com" {
			return fmt.Errorf("Bad email: expect: foo@foobar.com, got: %q", organizationMembership.Email)
		}
		return nil
	}
}

func testAccCheckTFETeamOrganizationMemberDestroy(s *terraform.State) error {
	tfeClient := testAccProvider.Meta().(*tfe.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "tfe_team_organization_member" {
			continue
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No instance ID is set")
		}

		// Get the team ID and organzation membership ID.
		teamID, organizationMembershipID, err := unpackTeamOrganizationMemberID(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error unpacking team organization member ID: %w", err)
		}

		organizationMemberships, err := tfeClient.TeamMembers.ListOrganizationMemberships(ctx, teamID)
		if errors.Is(err, tfe.ErrResourceNotFound) {
			return err
		}

		found := false
		for _, om := range organizationMemberships {
			if om.ID == organizationMembershipID {
				found = true
				break
			}
		}

		if found {
			return fmt.Errorf("Organization membership %s still exists", rs.Primary.ID)
		}
	}

	return nil
}

func testAccTFETeamOrganizationMember_basic(rInt int) string {
	return fmt.Sprintf(`
resource "tfe_organization" "foobar" {
  name  = "tst-terraform-%d"
  email = "admin@company.com"
}

resource "tfe_team" "foobar" {
  name         = "team-test"
  organization = tfe_organization.foobar.id
}

resource "tfe_organization_membership" "foobar" {
  organization = tfe_organization.foobar.id
	email = "foo@foobar.com"
}

resource "tfe_team_organization_member" "foobar" {
  team_id  = tfe_team.foobar.id
  organization_membership_id = tfe_organization_membership.foobar.id
}`, rInt)
}
