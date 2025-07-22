export interface Resource {
	Id: string;
	Mimetype: string;
	CreatedAt: string;
	Tags: string[];
}
export const DefaultResource = {
	Id: '',
	Mimetype: '',
	CreatedAt: '',
	Tags: []
};
